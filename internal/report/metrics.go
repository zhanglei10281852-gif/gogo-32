package report

import "gridflex/internal/model"

type Metrics struct {
	VehicleCount       int
	TargetMetCount     int
	WarningCount       int
	TotalChargedWh     int64
	TotalDischargedWh  int64
	AverageFairnessPPM int64
	PeakGridPowerW     int64
	MinimumGridPowerW  int64
}

func CalculateMetrics(plan model.Plan) Metrics {
	metrics := Metrics{
		VehicleCount: len(plan.Vehicles),
		WarningCount: len(plan.Warnings),
	}
	var fairnessTotal int64
	for _, vehicle := range plan.Vehicles {
		if vehicle.FinalSOCWh >= vehicle.TargetSOCWh {
			metrics.TargetMetCount++
		}
		metrics.TotalChargedWh += vehicle.ChargedBatteryWh
		metrics.TotalDischargedWh += vehicle.DischargedBatteryWh
		fairnessTotal += vehicle.FairnessScorePPM
	}
	if metrics.VehicleCount > 0 {
		metrics.AverageFairnessPPM = fairnessTotal / int64(metrics.VehicleCount)
	}
	for index, interval := range plan.Site {
		if index == 0 || interval.GridPowerW > metrics.PeakGridPowerW {
			metrics.PeakGridPowerW = interval.GridPowerW
		}
		if index == 0 || interval.GridPowerW < metrics.MinimumGridPowerW {
			metrics.MinimumGridPowerW = interval.GridPowerW
		}
	}
	return metrics
}

func IntervalCostSeries(settlement model.Settlement) []int64 {
	result := make([]int64, len(settlement.Intervals))
	for index, interval := range settlement.Intervals {
		result[index] = interval.NetCostMicro
	}
	return result
}

func IntervalCarbonSeries(settlement model.Settlement) []int64 {
	result := make([]int64, len(settlement.Intervals))
	for index, interval := range settlement.Intervals {
		result[index] = interval.CarbonGrams
	}
	return result
}

func TargetCompletionPPM(plan model.Plan) int64 {
	if len(plan.Vehicles) == 0 {
		return 0
	}
	var met int64
	for _, vehicle := range plan.Vehicles {
		if vehicle.FinalSOCWh >= vehicle.TargetSOCWh {
			met++
		}
	}
	return met * model.PartsPerMillion / int64(len(plan.Vehicles))
}
