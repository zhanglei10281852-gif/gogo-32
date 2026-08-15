package config

import (
	"fmt"

	"gridflex/internal/model"
)

func validateEvents(scenario model.Scenario, errs *validationErrors) {
	if len(scenario.Events) > 35040 {
		errs.Add("events exceeds maximum of 35040")
	}
	seen := make(map[string]struct{}, len(scenario.Events))
	previousEnd := -1
	for index, event := range scenario.Events {
		path := fmt.Sprintf("events[%d]", index)
		if event.ID == "" {
			errs.Add("%s.id is required", path)
		}
		if len(event.ID) > 128 {
			errs.Add("%s.id exceeds 128 characters", path)
		}
		for _, character := range event.ID {
			if character < 32 || character == 127 {
				errs.Add("%s.id contains a control character", path)
				break
			}
		}
		if _, exists := seen[event.ID]; exists {
			errs.Add("%s.id %q is duplicated", path, event.ID)
		}
		seen[event.ID] = struct{}{}
		if event.StartIndex < 0 {
			errs.Add("%s.start_index must be non-negative", path)
		}
		if event.EndIndex <= event.StartIndex {
			errs.Add("%s.end_index must exceed start_index", path)
		}
		if event.EndIndex > scenario.IntervalCount {
			errs.Add("%s.end_index exceeds interval_count", path)
		}
		validateCapacityLimits(path, event.ImportLimitW, event.ExportLimitW, errs)
		validRange := event.StartIndex >= 0 && event.EndIndex > event.StartIndex && event.EndIndex <= scenario.IntervalCount
		if validRange && previousEnd > event.StartIndex {
			errs.Add("%s overlaps an earlier capacity event", path)
		}
		if validRange && event.EndIndex > previousEnd {
			previousEnd = event.EndIndex
		}
	}
}

func validateCapacityLimits(path string, importLimitW, exportLimitW int64, errs *validationErrors) {
	if importLimitW <= 0 {
		errs.Add("%s.import_limit_w must be positive", path)
	}
	if exportLimitW < 0 {
		errs.Add("%s.export_limit_w must be non-negative", path)
	}
	if importLimitW > 10_000_000_000 {
		errs.Add("%s.import_limit_w exceeds supported range", path)
	}
	if exportLimitW > 10_000_000_000 {
		errs.Add("%s.export_limit_w exceeds supported range", path)
	}
}

func validateTariffs(scenario model.Scenario, errs *validationErrors) {
	if len(scenario.Tariffs) == 0 {
		errs.Add("tariffs must not be empty")
		return
	}
	var coverage []int
	if scenario.IntervalCount > 0 && scenario.IntervalCount <= 35040 {
		coverage = make([]int, scenario.IntervalCount)
	}
	for index, period := range scenario.Tariffs {
		path := fmt.Sprintf("tariffs[%d]", index)
		if period.StartIndex < 0 {
			errs.Add("%s.start_index must be non-negative", path)
		}
		if period.EndIndex <= period.StartIndex {
			errs.Add("%s.end_index must exceed start_index", path)
		}
		if period.EndIndex > scenario.IntervalCount {
			errs.Add("%s.end_index exceeds interval_count", path)
		}
		if period.ImportPriceMicroPerKWh < 0 {
			errs.Add("%s.import_price_micro_per_kwh must be non-negative", path)
		}
		if period.ExportPriceMicroPerKWh < 0 {
			errs.Add("%s.export_price_micro_per_kwh must be non-negative", path)
		}
		if period.CarbonGramsPerKWh < 0 {
			errs.Add("%s.carbon_g_per_kwh must be non-negative", path)
		}
		if coverage != nil {
			start := maxInt(0, period.StartIndex)
			end := minInt(scenario.IntervalCount, period.EndIndex)
			for slot := start; slot < end; slot++ {
				coverage[slot]++
			}
		}
	}
	for slot, count := range coverage {
		if count == 0 {
			errs.Add("tariffs do not cover interval %d", slot)
		}
		if count > 1 {
			errs.Add("tariffs overlap at interval %d", slot)
		}
	}
}

func validateForecast(scenario model.Scenario, errs *validationErrors) {
	if len(scenario.SiteForecast) != scenario.IntervalCount {
		errs.Add("site_forecast length %d must equal interval_count %d", len(scenario.SiteForecast), scenario.IntervalCount)
	}
	seen := make(map[int]struct{}, len(scenario.SiteForecast))
	for index, point := range scenario.SiteForecast {
		path := fmt.Sprintf("site_forecast[%d]", index)
		if point.Index < 0 || point.Index >= scenario.IntervalCount {
			errs.Add("%s.index is outside horizon", path)
		}
		if _, ok := seen[point.Index]; ok {
			errs.Add("%s.index %d is duplicated", path, point.Index)
		}
		seen[point.Index] = struct{}{}
		if point.BaseLoadW < 0 {
			errs.Add("%s.base_load_w must be non-negative", path)
		}
		if point.RenewableW < 0 {
			errs.Add("%s.renewable_w must be non-negative", path)
		}
		if point.BaseLoadW > 10_000_000_000 {
			errs.Add("%s.base_load_w exceeds supported range", path)
		}
		if point.RenewableW > 10_000_000_000 {
			errs.Add("%s.renewable_w exceeds supported range", path)
		}
	}
	for slot := 0; slot < scenario.IntervalCount; slot++ {
		if _, ok := seen[slot]; !ok {
			errs.Add("site_forecast is missing interval %d", slot)
		}
	}
}

func validateVehicles(scenario model.Scenario, errs *validationErrors) {
	if len(scenario.Vehicles) == 0 {
		errs.Add("vehicles must not be empty")
	}
	if len(scenario.Vehicles) > 10000 {
		errs.Add("vehicles exceeds maximum of 10000")
	}
	seen := make(map[string]struct{}, len(scenario.Vehicles))
	for index, vehicle := range scenario.Vehicles {
		path := fmt.Sprintf("vehicles[%d]", index)
		validateVehicleIdentity(path, vehicle, seen, errs)
		validateVehicleEnergy(path, vehicle, errs)
		validateVehiclePower(path, vehicle, errs)
		validateVehicleEfficiency(path, vehicle, errs)
		validateVehicleWindow(path, vehicle, scenario.IntervalCount, errs)
		validateVehicleReachability(path, vehicle, scenario.IntervalMinutes, errs)
	}
}

func validateVehicleIdentity(path string, vehicle model.Vehicle, seen map[string]struct{}, errs *validationErrors) {
	if vehicle.ID == "" {
		errs.Add("%s.id is required", path)
	}
	if len(vehicle.ID) > 128 {
		errs.Add("%s.id exceeds 128 characters", path)
	}
	if _, ok := seen[vehicle.ID]; ok {
		errs.Add("%s.id %q is duplicated", path, vehicle.ID)
	}
	seen[vehicle.ID] = struct{}{}
	for _, character := range vehicle.ID {
		if character < 32 || character == 127 {
			errs.Add("%s.id contains a control character", path)
			break
		}
	}
}

func validateVehicleEnergy(path string, vehicle model.Vehicle, errs *validationErrors) {
	if vehicle.CapacityWh <= 0 {
		errs.Add("%s.capacity_wh must be positive", path)
	}
	if vehicle.MinSOCWh < 0 {
		errs.Add("%s.min_soc_wh must be non-negative", path)
	}
	if vehicle.MaxSOCWh <= 0 {
		errs.Add("%s.max_soc_wh must be positive", path)
	}
	if vehicle.MaxSOCWh > vehicle.CapacityWh {
		errs.Add("%s.max_soc_wh exceeds capacity_wh", path)
	}
	if vehicle.MinSOCWh > vehicle.MaxSOCWh {
		errs.Add("%s.min_soc_wh exceeds max_soc_wh", path)
	}
	if vehicle.InitialSOCWh < vehicle.MinSOCWh || vehicle.InitialSOCWh > vehicle.MaxSOCWh {
		errs.Add("%s.initial_soc_wh is outside SOC bounds", path)
	}
	if vehicle.MinDepartureSOCWh < vehicle.MinSOCWh || vehicle.MinDepartureSOCWh > vehicle.MaxSOCWh {
		errs.Add("%s.min_departure_soc_wh is outside SOC bounds", path)
	}
}

func validateVehiclePower(path string, vehicle model.Vehicle, errs *validationErrors) {
	if vehicle.MaxChargeW <= 0 {
		errs.Add("%s.max_charge_w must be positive", path)
	}
	if vehicle.MaxDischargeW < 0 {
		errs.Add("%s.max_discharge_w must be non-negative", path)
	}
	if vehicle.AllowDischarge && vehicle.MaxDischargeW == 0 {
		errs.Add("%s.max_discharge_w must be positive when discharge is allowed", path)
	}
	if !vehicle.AllowDischarge && vehicle.MaxDischargeW != 0 {
		errs.Add("%s.max_discharge_w must be zero when discharge is disabled", path)
	}
	if vehicle.FairnessWeight <= 0 {
		errs.Add("%s.fairness_weight must be positive", path)
	}
}

func validateVehicleEfficiency(path string, vehicle model.Vehicle, errs *validationErrors) {
	if vehicle.ChargeEfficiencyPPM <= 0 || vehicle.ChargeEfficiencyPPM > model.PartsPerMillion {
		errs.Add("%s.charge_efficiency_ppm must be in 1..1000000", path)
	}
	if vehicle.DischargeEfficiencyPPM <= 0 || vehicle.DischargeEfficiencyPPM > model.PartsPerMillion {
		errs.Add("%s.discharge_efficiency_ppm must be in 1..1000000", path)
	}
}

func validateVehicleWindow(path string, vehicle model.Vehicle, intervalCount int, errs *validationErrors) {
	if vehicle.ArrivalIndex < 0 {
		errs.Add("%s.arrival_index must be non-negative", path)
	}
	if vehicle.DepartureIndex <= vehicle.ArrivalIndex {
		errs.Add("%s.departure_index must exceed arrival_index", path)
	}
	if vehicle.DepartureIndex > intervalCount {
		errs.Add("%s.departure_index exceeds interval_count", path)
	}
}

func validateVehicleReachability(path string, vehicle model.Vehicle, intervalMinutes int64, errs *validationErrors) {
	if intervalMinutes <= 0 || vehicle.MaxChargeW <= 0 || vehicle.ChargeEfficiencyPPM <= 0 {
		return
	}
	available := int64(vehicle.DepartureIndex - vehicle.ArrivalIndex)
	gridWh, err := model.PowerToEnergyFloor(vehicle.MaxChargeW, intervalMinutes*available)
	if err != nil {
		errs.Add("%s charge reachability overflow", path)
		return
	}
	batteryWh, err := model.ApplyChargeEfficiency(gridWh, vehicle.ChargeEfficiencyPPM)
	if err != nil {
		errs.Add("%s charge reachability overflow", path)
		return
	}
	if vehicle.InitialSOCWh+batteryWh < vehicle.MinDepartureSOCWh {
		errs.Add("%s departure target is unreachable at maximum charge power", path)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
