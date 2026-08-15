package constraint

import (
	"fmt"
	"sort"

	"gridflex/internal/model"
)

type Violation struct {
	Code          string `json:"code"`
	IntervalIndex int    `json:"interval_index"`
	VehicleID     string `json:"vehicle_id,omitempty"`
	Message       string `json:"message"`
}

func ValidatePlan(scenario model.Scenario, plan model.Plan) []Violation {
	violations := make([]Violation, 0)
	if plan.ScenarioID != scenario.ScenarioID {
		violations = append(violations, Violation{Code: "scenario_mismatch", IntervalIndex: -1, Message: "plan scenario ID does not match input"})
	}
	vehicles := make(map[string]model.Vehicle, len(scenario.Vehicles))
	soc := make(map[string]int64, len(scenario.Vehicles))
	for _, vehicle := range scenario.Vehicles {
		vehicles[vehicle.ID] = vehicle
		soc[vehicle.ID] = vehicle.InitialSOCWh
	}
	actions := append([]model.Action(nil), plan.Actions...)
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].IntervalIndex != actions[j].IntervalIndex {
			return actions[i].IntervalIndex < actions[j].IntervalIndex
		}
		return actions[i].VehicleID < actions[j].VehicleID
	})
	fleetPower := make([]int64, scenario.IntervalCount)
	seenAction := make(map[string]struct{})
	for _, action := range actions {
		vehicle, ok := vehicles[action.VehicleID]
		if !ok {
			violations = append(violations, Violation{Code: "unknown_vehicle", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "action references unknown vehicle"})
			continue
		}
		key := fmt.Sprintf("%d/%s", action.IntervalIndex, action.VehicleID)
		if _, ok := seenAction[key]; ok {
			violations = append(violations, Violation{Code: "duplicate_action", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "multiple actions for vehicle in interval"})
		}
		seenAction[key] = struct{}{}
		if action.IntervalIndex < 0 || action.IntervalIndex >= scenario.IntervalCount {
			violations = append(violations, Violation{Code: "interval_range", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "action interval outside horizon"})
			continue
		}
		if action.IntervalIndex < vehicle.ArrivalIndex || action.IntervalIndex >= vehicle.DepartureIndex {
			violations = append(violations, Violation{Code: "availability", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "action outside availability window"})
		}
		if action.PowerW > vehicle.MaxChargeW {
			violations = append(violations, Violation{Code: "charge_power", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "charge power exceeds vehicle limit"})
		}
		if action.PowerW < -vehicle.MaxDischargeW {
			violations = append(violations, Violation{Code: "discharge_power", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "discharge power exceeds vehicle limit"})
		}
		if action.PowerW < 0 && !vehicle.AllowDischarge {
			violations = append(violations, Violation{Code: "discharge_disabled", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "vehicle discharge is disabled"})
		}
		if action.SOCBeforeWh != soc[action.VehicleID] {
			violations = append(violations, Violation{Code: "soc_continuity", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "SOC before does not match prior state"})
		}
		if action.SOCAfterWh != action.SOCBeforeWh+action.BatteryDeltaWh {
			violations = append(violations, Violation{Code: "soc_delta", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "SOC delta is inconsistent"})
		}
		if action.SOCAfterWh < vehicle.MinSOCWh || action.SOCAfterWh > vehicle.MaxSOCWh {
			violations = append(violations, Violation{Code: "soc_bounds", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "SOC after action outside bounds"})
		}
		violations = append(violations, validateEfficiency(scenario, vehicle, action)...)
		soc[action.VehicleID] = action.SOCAfterWh
		fleetPower[action.IntervalIndex] += action.PowerW
	}
	violations = append(violations, validateSite(scenario, fleetPower)...)
	for _, vehicle := range scenario.Vehicles {
		if soc[vehicle.ID] < vehicle.MinDepartureSOCWh {
			violations = append(violations, Violation{Code: "departure_soc", IntervalIndex: vehicle.DepartureIndex, VehicleID: vehicle.ID, Message: "final SOC below departure target"})
		}
	}
	violations = append(violations, validateFairness(scenario, plan)...)
	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].IntervalIndex != violations[j].IntervalIndex {
			return violations[i].IntervalIndex < violations[j].IntervalIndex
		}
		if violations[i].VehicleID != violations[j].VehicleID {
			return violations[i].VehicleID < violations[j].VehicleID
		}
		return violations[i].Code < violations[j].Code
	})
	return violations
}

func validateEfficiency(scenario model.Scenario, vehicle model.Vehicle, action model.Action) []Violation {
	result := make([]Violation, 0, 1)
	if action.PowerW > 0 {
		expectedGrid, err := model.PowerToEnergyFloor(action.PowerW, scenario.IntervalMinutes)
		if err != nil || expectedGrid != action.GridEnergyWh {
			result = append(result, Violation{Code: "grid_energy", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "charge grid energy inconsistent with power"})
			return result
		}
		expectedBattery, err := model.ApplyChargeEfficiency(action.GridEnergyWh, vehicle.ChargeEfficiencyPPM)
		if err != nil || expectedBattery != action.BatteryDeltaWh {
			result = append(result, Violation{Code: "charge_efficiency", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "charge battery delta inconsistent with efficiency"})
		}
	}
	if action.PowerW < 0 {
		expectedGrid, err := model.PowerToEnergyFloor(-action.PowerW, scenario.IntervalMinutes)
		if err != nil || expectedGrid != action.GridEnergyWh {
			result = append(result, Violation{Code: "grid_energy", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "discharge grid energy inconsistent with power"})
			return result
		}
		expectedBattery, err := model.BatteryForExportCeil(action.GridEnergyWh, vehicle.DischargeEfficiencyPPM)
		if err != nil || -expectedBattery != action.BatteryDeltaWh {
			result = append(result, Violation{Code: "discharge_efficiency", IntervalIndex: action.IntervalIndex, VehicleID: action.VehicleID, Message: "discharge battery delta inconsistent with efficiency"})
		}
	}
	return result
}
