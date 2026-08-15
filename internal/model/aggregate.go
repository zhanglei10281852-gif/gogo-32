package model

import "sort"

type EnergyTotals struct {
	GridImportWh        int64
	GridExportWh        int64
	BatteryChargedWh    int64
	BatteryDischargedWh int64
	ChargeLossWh        int64
	DischargeLossWh     int64
}

func AggregateActions(actions []Action) EnergyTotals {
	var result EnergyTotals
	for _, action := range actions {
		if action.PowerW > 0 {
			result.GridImportWh += action.GridEnergyWh
			result.BatteryChargedWh += action.BatteryDeltaWh
			result.ChargeLossWh += action.GridEnergyWh - action.BatteryDeltaWh
		}
		if action.PowerW < 0 {
			batteryWh := -action.BatteryDeltaWh
			result.GridExportWh += action.GridEnergyWh
			result.BatteryDischargedWh += batteryWh
			result.DischargeLossWh += batteryWh - action.GridEnergyWh
		}
	}
	return result
}

func ActionsByVehicle(actions []Action) map[string][]Action {
	result := make(map[string][]Action)
	for _, action := range actions {
		result[action.VehicleID] = append(result[action.VehicleID], action)
	}
	for id := range result {
		sort.SliceStable(result[id], func(i, j int) bool {
			if result[id][i].IntervalIndex != result[id][j].IntervalIndex {
				return result[id][i].IntervalIndex < result[id][j].IntervalIndex
			}
			return result[id][i].PowerW < result[id][j].PowerW
		})
	}
	return result
}

func ActionsByInterval(actions []Action, count int) [][]Action {
	if count < 0 {
		count = 0
	}
	result := make([][]Action, count)
	for _, action := range actions {
		if action.IntervalIndex < 0 || action.IntervalIndex >= count {
			continue
		}
		result[action.IntervalIndex] = append(result[action.IntervalIndex], action)
	}
	for index := range result {
		sort.SliceStable(result[index], func(i, j int) bool {
			return result[index][i].VehicleID < result[index][j].VehicleID
		})
	}
	return result
}

func VehicleSummaryByID(summaries []VehicleSummary) map[string]VehicleSummary {
	result := make(map[string]VehicleSummary, len(summaries))
	for _, summary := range summaries {
		result[summary.VehicleID] = summary
	}
	return result
}

func SortedVehicleIDs(vehicles []Vehicle) []string {
	result := make([]string, len(vehicles))
	for index, vehicle := range vehicles {
		result[index] = vehicle.ID
	}
	sort.Strings(result)
	return result
}
