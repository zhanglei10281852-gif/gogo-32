package constraint

import (
	"fmt"
	"sort"

	"gridflex/internal/model"
)

type CapacityDiagnostic struct {
	IntervalIndex        int
	BaseNetW             int64
	FleetPowerW          int64
	GridPowerW           int64
	ImportHeadroomW      int64
	ExportHeadroomW      int64
	ImportUtilizationPPM int64
	ExportUtilizationPPM int64
}

type VehicleDiagnostic struct {
	VehicleID            string
	InitialSOCWh         int64
	FinalSOCWh           int64
	TargetSOCWh          int64
	TargetSurplusWh      int64
	MinimumObservedSOCWh int64
	MaximumObservedSOCWh int64
	ChargeActionCount    int
	DischargeActionCount int
	AvailableActionCount int
	WeightedProgressPPM  int64
}

func CapacityDiagnostics(scenario model.Scenario, plan model.Plan) ([]CapacityDiagnostic, error) {
	if len(plan.Site) != scenario.IntervalCount {
		return nil, fmt.Errorf("site interval count %d does not match scenario %d", len(plan.Site), scenario.IntervalCount)
	}
	result := make([]CapacityDiagnostic, scenario.IntervalCount)
	for index, site := range plan.Site {
		if site.Index != index {
			return nil, fmt.Errorf("site interval at position %d has index %d", index, site.Index)
		}
		capacity := model.SiteCapacityAt(scenario, index)
		importHeadroom := capacity.ImportLimitW - site.GridPowerW
		if importHeadroom < 0 {
			importHeadroom = 0
		}
		exportHeadroom := capacity.ExportLimitW + site.GridPowerW
		if exportHeadroom < 0 {
			exportHeadroom = 0
		}
		importUtilization := int64(0)
		if site.GridPowerW > 0 && capacity.ImportLimitW > 0 {
			importUtilization, _ = model.MulDivFloor(site.GridPowerW, model.PartsPerMillion, capacity.ImportLimitW)
		}
		exportUtilization := int64(0)
		if site.GridPowerW < 0 && capacity.ExportLimitW > 0 {
			exportUtilization, _ = model.MulDivFloor(-site.GridPowerW, model.PartsPerMillion, capacity.ExportLimitW)
		}
		result[index] = CapacityDiagnostic{
			IntervalIndex:        index,
			BaseNetW:             site.BaseNetW,
			FleetPowerW:          site.FleetPowerW,
			GridPowerW:           site.GridPowerW,
			ImportHeadroomW:      importHeadroom,
			ExportHeadroomW:      exportHeadroom,
			ImportUtilizationPPM: importUtilization,
			ExportUtilizationPPM: exportUtilization,
		}
	}
	return result, nil
}

func VehicleDiagnostics(scenario model.Scenario, plan model.Plan) ([]VehicleDiagnostic, error) {
	vehicles := make(map[string]model.Vehicle, len(scenario.Vehicles))
	for _, vehicle := range scenario.Vehicles {
		vehicles[vehicle.ID] = vehicle
	}
	diagnostics := make(map[string]*VehicleDiagnostic, len(vehicles))
	for id, vehicle := range vehicles {
		diagnostics[id] = &VehicleDiagnostic{
			VehicleID:            id,
			InitialSOCWh:         vehicle.InitialSOCWh,
			FinalSOCWh:           vehicle.InitialSOCWh,
			TargetSOCWh:          vehicle.MinDepartureSOCWh,
			MinimumObservedSOCWh: vehicle.InitialSOCWh,
			MaximumObservedSOCWh: vehicle.InitialSOCWh,
			AvailableActionCount: vehicle.DepartureIndex - vehicle.ArrivalIndex,
		}
	}
	actions := append([]model.Action(nil), plan.Actions...)
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].IntervalIndex != actions[j].IntervalIndex {
			return actions[i].IntervalIndex < actions[j].IntervalIndex
		}
		return actions[i].VehicleID < actions[j].VehicleID
	})
	for _, action := range actions {
		diagnostic, ok := diagnostics[action.VehicleID]
		if !ok {
			return nil, fmt.Errorf("action references unknown vehicle %q", action.VehicleID)
		}
		if action.PowerW > 0 {
			diagnostic.ChargeActionCount++
		}
		if action.PowerW < 0 {
			diagnostic.DischargeActionCount++
		}
		diagnostic.FinalSOCWh = action.SOCAfterWh
		if action.SOCAfterWh < diagnostic.MinimumObservedSOCWh {
			diagnostic.MinimumObservedSOCWh = action.SOCAfterWh
		}
		if action.SOCAfterWh > diagnostic.MaximumObservedSOCWh {
			diagnostic.MaximumObservedSOCWh = action.SOCAfterWh
		}
	}
	result := make([]VehicleDiagnostic, 0, len(diagnostics))
	for id, diagnostic := range diagnostics {
		vehicle := vehicles[id]
		diagnostic.TargetSurplusWh = diagnostic.FinalSOCWh - diagnostic.TargetSOCWh
		need := vehicle.MinDepartureSOCWh - vehicle.InitialSOCWh
		if need <= 0 {
			diagnostic.WeightedProgressPPM = model.PartsPerMillion
		} else {
			progress := diagnostic.FinalSOCWh - vehicle.InitialSOCWh
			if progress < 0 {
				progress = 0
			}
			diagnostic.WeightedProgressPPM, _ = model.MulDivFloor(progress, model.PartsPerMillion, need*vehicle.FairnessWeight)
		}
		result = append(result, *diagnostic)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].VehicleID < result[j].VehicleID
	})
	return result, nil
}

func ViolationCounts(violations []Violation) map[string]int {
	result := make(map[string]int)
	for _, violation := range violations {
		result[violation.Code]++
	}
	return result
}

func FirstViolation(violations []Violation) (Violation, bool) {
	if len(violations) == 0 {
		return Violation{}, false
	}
	ordered := append([]Violation(nil), violations...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].IntervalIndex != ordered[j].IntervalIndex {
			return ordered[i].IntervalIndex < ordered[j].IntervalIndex
		}
		if ordered[i].VehicleID != ordered[j].VehicleID {
			return ordered[i].VehicleID < ordered[j].VehicleID
		}
		return ordered[i].Code < ordered[j].Code
	})
	return ordered[0], true
}
