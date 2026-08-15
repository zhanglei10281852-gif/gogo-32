package dispatch

import (
	"fmt"
	"time"

	"gridflex/internal/forecast"
	"gridflex/internal/model"
	"gridflex/internal/tariff"
)

func BuildBaseline(scenario model.Scenario) (model.Baseline, error) {
	series, err := forecast.FromScenario(scenario)
	if err != nil {
		return model.Baseline{}, err
	}
	rates, err := tariff.FromScenario(scenario)
	if err != nil {
		return model.Baseline{}, err
	}
	intervals := make([]model.BaselineInterval, scenario.IntervalCount)
	for index := 0; index < scenario.IntervalCount; index++ {
		point, err := series.At(index)
		if err != nil {
			return model.Baseline{}, fmt.Errorf("baseline forecast: %w", err)
		}
		rate, err := rates.At(index)
		if err != nil {
			return model.Baseline{}, fmt.Errorf("baseline tariff: %w", err)
		}
		intervals[index] = model.BaselineInterval{
			Index:                  index,
			BaseLoadW:              point.BaseLoadW,
			RenewableW:             point.RenewableW,
			NetSiteW:               point.NetLoadW,
			ImportPriceMicroPerKWh: rate.ImportPriceMicroPerKWh,
			ExportPriceMicroPerKWh: rate.ExportPriceMicroPerKWh,
			CarbonGramsPerKWh:      rate.CarbonGramsPerKWh,
		}
	}
	return model.Baseline{
		ScenarioID: scenario.ScenarioID,
		CreatedAt:  deterministicTime(scenario),
		Intervals:  intervals,
	}, nil
}

func deterministicTime(scenario model.Scenario) time.Time {
	return scenario.Start.UTC()
}
