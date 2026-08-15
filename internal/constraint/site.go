package constraint

import "gridflex/internal/model"

func validateSite(scenario model.Scenario, fleetPower []int64) []Violation {
	result := make([]Violation, 0)
	forecast := make(map[int]model.ForecastPoint, len(scenario.SiteForecast))
	for _, point := range scenario.SiteForecast {
		forecast[point.Index] = point
	}
	for index := 0; index < scenario.IntervalCount; index++ {
		point := forecast[index]
		gridPower := point.BaseLoadW - point.RenewableW + fleetPower[index]
		capacity := model.SiteCapacityAt(scenario, index)
		if gridPower > capacity.ImportLimitW {
			result = append(result, Violation{Code: "site_import", IntervalIndex: index, Message: "site import capacity exceeded"})
		}
		if gridPower < -capacity.ExportLimitW {
			result = append(result, Violation{Code: "site_export", IntervalIndex: index, Message: "site export capacity exceeded"})
		}
	}
	return result
}

func validateFairness(scenario model.Scenario, plan model.Plan) []Violation {
	if scenario.Options.FairnessToleranceWh == 0 || len(plan.Vehicles) < 2 {
		return nil
	}
	eligible := make([]model.VehicleSummary, 0, len(plan.Vehicles))
	vehicleByID := make(map[string]model.Vehicle, len(scenario.Vehicles))
	for _, vehicle := range scenario.Vehicles {
		vehicleByID[vehicle.ID] = vehicle
	}
	for _, summary := range plan.Vehicles {
		vehicle := vehicleByID[summary.VehicleID]
		if vehicle.MinDepartureSOCWh > vehicle.InitialSOCWh {
			eligible = append(eligible, summary)
		}
	}
	result := make([]Violation, 0)
	for left := 0; left < len(eligible); left++ {
		for right := left + 1; right < len(eligible); right++ {
			leftVehicle := vehicleByID[eligible[left].VehicleID]
			rightVehicle := vehicleByID[eligible[right].VehicleID]
			leftProgress := eligible[left].FinalSOCWh - leftVehicle.InitialSOCWh
			rightProgress := eligible[right].FinalSOCWh - rightVehicle.InitialSOCWh
			leftWeighted := leftProgress * rightVehicle.FairnessWeight
			rightWeighted := rightProgress * leftVehicle.FairnessWeight
			difference := model.Abs64(leftWeighted - rightWeighted)
			tolerance := scenario.Options.FairnessToleranceWh * leftVehicle.FairnessWeight * rightVehicle.FairnessWeight
			if difference > tolerance && eligible[left].FinalSOCWh < eligible[left].TargetSOCWh && eligible[right].FinalSOCWh < eligible[right].TargetSOCWh {
				result = append(result, Violation{Code: "fairness", IntervalIndex: -1, VehicleID: eligible[left].VehicleID, Message: "weighted charging progress exceeds fairness tolerance"})
			}
		}
	}
	return result
}

func IsFeasible(scenario model.Scenario, plan model.Plan) bool {
	return len(ValidatePlan(scenario, plan)) == 0
}
