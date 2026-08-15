package dispatch

import (
	"errors"
	"fmt"
	"sort"

	"gridflex/internal/constraint"
	"gridflex/internal/fleet"
	"gridflex/internal/forecast"
	"gridflex/internal/model"
	"gridflex/internal/tariff"
	"gridflex/internal/timegrid"
)

type Planner struct {
	scenario model.Scenario
	grid     timegrid.Grid
	forecast forecast.Series
	tariffs  tariff.Schedule
}

func New(scenario model.Scenario) (*Planner, error) {
	grid, err := timegrid.FromScenario(scenario)
	if err != nil {
		return nil, fmt.Errorf("create grid: %w", err)
	}
	series, err := forecast.FromScenario(scenario)
	if err != nil {
		return nil, fmt.Errorf("create forecast: %w", err)
	}
	rates, err := tariff.FromScenario(scenario)
	if err != nil {
		return nil, fmt.Errorf("create tariff: %w", err)
	}
	return &Planner{scenario: scenario, grid: grid, forecast: series, tariffs: rates}, nil
}

func (p *Planner) Plan() (model.Plan, error) {
	working, err := fleet.New(p.scenario.Vehicles)
	if err != nil {
		return model.Plan{}, err
	}
	plan := model.Plan{
		ScenarioID: p.scenario.ScenarioID,
		CreatedAt:  deterministicTime(p.scenario),
		Feasible:   true,
		Actions:    make([]model.Action, 0),
		Site:       make([]model.SiteInterval, p.scenario.IntervalCount),
		Warnings:   make([]string, 0),
	}
	for index := 0; index < p.scenario.IntervalCount; index++ {
		if err := working.AssertDeparture(index); err != nil {
			plan.Warnings = append(plan.Warnings, err.Error())
		}
		site, actions, err := p.planInterval(index, working)
		if err != nil {
			return model.Plan{}, fmt.Errorf("plan interval %d: %w", index, err)
		}
		plan.Site[index] = site
		plan.Actions = append(plan.Actions, actions...)
	}
	if err := working.AssertDeparture(p.scenario.IntervalCount); err != nil {
		plan.Warnings = append(plan.Warnings, err.Error())
	}
	plan.Vehicles = working.Summaries()
	if !working.AllDepartureTargetsMet() {
		plan.Feasible = false
		plan.Warnings = append(plan.Warnings, "one or more departure SOC targets were not met")
	}
	sort.SliceStable(plan.Actions, func(i, j int) bool {
		if plan.Actions[i].IntervalIndex != plan.Actions[j].IntervalIndex {
			return plan.Actions[i].IntervalIndex < plan.Actions[j].IntervalIndex
		}
		return plan.Actions[i].VehicleID < plan.Actions[j].VehicleID
	})
	sort.SliceStable(plan.Vehicles, func(i, j int) bool {
		return plan.Vehicles[i].VehicleID < plan.Vehicles[j].VehicleID
	})
	violations := constraint.ValidatePlan(p.scenario, plan)
	if len(violations) != 0 {
		plan.Feasible = false
		for _, violation := range violations {
			plan.Warnings = append(plan.Warnings, formatViolation(violation))
		}
	}
	sort.Strings(plan.Warnings)
	plan.Warnings = uniqueStrings(plan.Warnings)
	return plan, nil
}

func (p *Planner) planInterval(index int, working *fleet.Fleet) (model.SiteInterval, []model.Action, error) {
	point, err := p.forecast.At(index)
	if err != nil {
		return model.SiteInterval{}, nil, err
	}
	capacity := model.SiteCapacityAt(p.scenario, index)
	baseNetW := point.NetLoadW
	chargeHeadroomW := capacity.ImportLimitW - baseNetW
	if chargeHeadroomW < 0 {
		chargeHeadroomW = 0
	}
	actions := make([]model.Action, 0)
	priorities, err := working.ChargePriorities(index, p.scenario.IntervalMinutes, p.scenario.Options.ReserveMarginWh)
	if err != nil {
		return model.SiteInterval{}, nil, err
	}
	for _, priority := range priorities {
		if chargeHeadroomW <= 0 {
			break
		}
		state, err := working.State(priority.VehicleID)
		if err != nil {
			return model.SiteInterval{}, nil, err
		}
		powerW, err := p.selectChargePower(priority, state, chargeHeadroomW)
		if err != nil {
			return model.SiteInterval{}, nil, err
		}
		if powerW <= 0 {
			continue
		}
		action, err := p.applyCharge(index, priority.VehicleID, powerW, working)
		if err != nil {
			return model.SiteInterval{}, nil, err
		}
		actions = append(actions, action)
		chargeHeadroomW -= powerW
	}
	fleetPowerW := sumActionPower(actions)
	if p.scenario.Options.EnableExport {
		dischargeActions, err := p.planDischarge(index, capacity.ExportLimitW, baseNetW+fleetPowerW, working, actionVehicleSet(actions))
		if err != nil {
			return model.SiteInterval{}, nil, err
		}
		actions = append(actions, dischargeActions...)
		fleetPowerW = sumActionPower(actions)
	}
	gridPowerW := baseNetW + fleetPowerW
	importWh, exportWh, err := p.gridEnergy(gridPowerW)
	if err != nil {
		return model.SiteInterval{}, nil, err
	}
	sort.SliceStable(actions, func(i, j int) bool {
		return actions[i].VehicleID < actions[j].VehicleID
	})
	return model.SiteInterval{
		Index:       index,
		BaseNetW:    baseNetW,
		FleetPowerW: fleetPowerW,
		GridPowerW:  gridPowerW,
		ImportWh:    importWh,
		ExportWh:    exportWh,
	}, actions, nil
}

func (p *Planner) selectChargePower(priority fleet.Priority, state fleet.State, headroomW int64) (int64, error) {
	maximum := model.Min64(state.Vehicle.MaxChargeW, headroomW)
	if maximum <= 0 {
		return 0, nil
	}
	batteryCapacity := state.Vehicle.MaxSOCWh - state.SOCWh
	need := model.Min64(priority.NeedWh, batteryCapacity)
	if need <= 0 {
		return 0, nil
	}
	gridNeed, err := model.GridForChargeCeil(need, state.Vehicle.ChargeEfficiencyPPM)
	if err != nil {
		return 0, err
	}
	needPower, err := model.EnergyToPowerFloor(gridNeed, p.scenario.IntervalMinutes)
	if err != nil {
		return 0, err
	}
	if needPower < 1 {
		needPower = 1
	}
	power := model.Min64(maximum, needPower)
	if priority.RemainingIntervals > 1 {
		power = model.Min64(power, model.Max64(priority.RequiredPowerW, 1))
	}
	for power > 0 {
		gridWh, err := model.PowerToEnergyFloor(power, p.scenario.IntervalMinutes)
		if err != nil {
			return 0, err
		}
		batteryWh, err := model.ApplyChargeEfficiency(gridWh, state.Vehicle.ChargeEfficiencyPPM)
		if err != nil {
			return 0, err
		}
		if batteryWh <= batteryCapacity && batteryWh <= need {
			break
		}
		power--
	}
	return power, nil
}

func (p *Planner) applyCharge(index int, id string, powerW int64, working *fleet.Fleet) (model.Action, error) {
	gridWh, err := model.PowerToEnergyFloor(powerW, p.scenario.IntervalMinutes)
	if err != nil {
		return model.Action{}, err
	}
	if gridWh == 0 {
		return model.Action{}, errors.New("charge power rounds to zero energy")
	}
	batteryWh, before, after, err := working.Charge(id, gridWh)
	if err != nil {
		return model.Action{}, err
	}
	return model.Action{
		IntervalIndex:  index,
		VehicleID:      id,
		PowerW:         powerW,
		GridEnergyWh:   gridWh,
		BatteryDeltaWh: batteryWh,
		SOCBeforeWh:    before,
		SOCAfterWh:     after,
		Reason:         "departure-target weighted-fair charge",
	}, nil
}
func (p *Planner) planDischarge(index int, exportLimitW, currentGridW int64, working *fleet.Fleet, already map[string]struct{}) ([]model.Action, error) {
	attractive, err := p.tariffs.IsExportAttractive(index)
	if err != nil {
		return nil, err
	}
	if !attractive || exportLimitW <= 0 {
		return nil, nil
	}
	exportHeadroomW := exportLimitW + currentGridW
	if exportHeadroomW <= 0 {
		return nil, nil
	}
	priorities, err := working.DischargePriorities(index, p.scenario.Options.ReserveMarginWh)
	if err != nil {
		return nil, err
	}
	actions := make([]model.Action, 0)
	for _, priority := range priorities {
		if exportHeadroomW <= 0 {
			break
		}
		if _, exists := already[priority.VehicleID]; exists {
			continue
		}
		state, err := working.State(priority.VehicleID)
		if err != nil {
			return nil, err
		}
		maximum := model.Min64(state.Vehicle.MaxDischargeW, exportHeadroomW)
		batteryAvailable, err := working.DischargeAvailableWh(priority.VehicleID, p.scenario.Options.ReserveMarginWh)
		if err != nil {
			return nil, err
		}
		gridAvailable, err := model.GridFromDischargeFloor(batteryAvailable, state.Vehicle.DischargeEfficiencyPPM)
		if err != nil {
			return nil, err
		}
		powerAvailable, err := model.EnergyToPowerFloor(gridAvailable, p.scenario.IntervalMinutes)
		if err != nil {
			return nil, err
		}
		powerW := model.Min64(maximum, powerAvailable)
		for powerW > 0 {
			gridWh, err := model.PowerToEnergyFloor(powerW, p.scenario.IntervalMinutes)
			if err != nil {
				return nil, err
			}
			batteryWh, err := model.BatteryForExportCeil(gridWh, state.Vehicle.DischargeEfficiencyPPM)
			if err != nil {
				return nil, err
			}
			if batteryWh <= batteryAvailable {
				break
			}
			powerW--
		}
		if powerW <= 0 {
			continue
		}
		action, err := p.applyDischarge(index, priority.VehicleID, powerW, working)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
		exportHeadroomW -= powerW
	}
	return actions, nil
}

func (p *Planner) applyDischarge(index int, id string, powerW int64, working *fleet.Fleet) (model.Action, error) {
	gridWh, err := model.PowerToEnergyFloor(powerW, p.scenario.IntervalMinutes)
	if err != nil {
		return model.Action{}, err
	}
	if gridWh == 0 {
		return model.Action{}, errors.New("discharge power rounds to zero energy")
	}
	batteryWh, before, after, err := working.Discharge(id, gridWh)
	if err != nil {
		return model.Action{}, err
	}
	return model.Action{
		IntervalIndex:  index,
		VehicleID:      id,
		PowerW:         -powerW,
		GridEnergyWh:   gridWh,
		BatteryDeltaWh: -batteryWh,
		SOCBeforeWh:    before,
		SOCAfterWh:     after,
		Reason:         "price-responsive constrained export",
	}, nil
}

func (p *Planner) gridEnergy(gridPowerW int64) (importWh int64, exportWh int64, err error) {
	if gridPowerW >= 0 {
		importWh, err = model.PowerToEnergyFloor(gridPowerW, p.scenario.IntervalMinutes)
		return importWh, 0, err
	}
	exportWh, err = model.PowerToEnergyFloor(-gridPowerW, p.scenario.IntervalMinutes)
	return 0, exportWh, err
}

func sumActionPower(actions []model.Action) int64 {
	var total int64
	for _, action := range actions {
		total += action.PowerW
	}
	return total
}

func actionVehicleSet(actions []model.Action) map[string]struct{} {
	result := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		result[action.VehicleID] = struct{}{}
	}
	return result
}

func formatViolation(violation constraint.Violation) string {
	if violation.VehicleID != "" {
		return fmt.Sprintf("%s at interval %d for %s: %s", violation.Code, violation.IntervalIndex, violation.VehicleID, violation.Message)
	}
	return fmt.Sprintf("%s at interval %d: %s", violation.Code, violation.IntervalIndex, violation.Message)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	result := values[:1]
	for index := 1; index < len(values); index++ {
		if values[index] != values[index-1] {
			result = append(result, values[index])
		}
	}
	return result
}
