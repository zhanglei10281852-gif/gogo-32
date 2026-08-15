package settlement

import (
	"fmt"

	"gridflex/internal/model"
	"gridflex/internal/tariff"
)

type Engine struct {
	scenario model.Scenario
	tariffs  tariff.Schedule
}

func New(scenario model.Scenario) (*Engine, error) {
	rates, err := tariff.FromScenario(scenario)
	if err != nil {
		return nil, fmt.Errorf("settlement tariff: %w", err)
	}
	return &Engine{scenario: scenario, tariffs: rates}, nil
}

func (e *Engine) Settle(baseline model.Baseline, plan model.Plan) (model.Settlement, error) {
	if baseline.ScenarioID != e.scenario.ScenarioID {
		return model.Settlement{}, fmt.Errorf("baseline scenario %q does not match %q", baseline.ScenarioID, e.scenario.ScenarioID)
	}
	if plan.ScenarioID != e.scenario.ScenarioID {
		return model.Settlement{}, fmt.Errorf("plan scenario %q does not match %q", plan.ScenarioID, e.scenario.ScenarioID)
	}
	if len(baseline.Intervals) != e.scenario.IntervalCount {
		return model.Settlement{}, fmt.Errorf("baseline has %d intervals, expected %d", len(baseline.Intervals), e.scenario.IntervalCount)
	}
	if len(plan.Site) != e.scenario.IntervalCount {
		return model.Settlement{}, fmt.Errorf("plan has %d site intervals, expected %d", len(plan.Site), e.scenario.IntervalCount)
	}
	result := model.Settlement{
		ScenarioID: e.scenario.ScenarioID,
		CreatedAt:  e.scenario.Start.UTC(),
		Intervals:  make([]model.SettlementInterval, e.scenario.IntervalCount),
	}
	for index := 0; index < e.scenario.IntervalCount; index++ {
		interval, err := e.settleInterval(index, baseline.Intervals[index], plan.Site[index])
		if err != nil {
			return model.Settlement{}, err
		}
		result.Intervals[index] = interval
		accumulateInterval(&result, interval)
	}
	vehicles, err := e.settleVehicles(plan.Actions)
	if err != nil {
		return model.Settlement{}, err
	}
	result.Vehicles = vehicles
	result.NetCostMicro = result.TotalImportCostMicro - result.TotalExportRevenueMicro
	result.SavingsMicro = result.BaselineCostMicro - result.NetCostMicro
	result.CarbonDeltaGrams = result.CarbonGrams - result.BaselineCarbonGrams
	return result, nil
}

func (e *Engine) settleInterval(index int, baseline model.BaselineInterval, site model.SiteInterval) (model.SettlementInterval, error) {
	if baseline.Index != index || site.Index != index {
		return model.SettlementInterval{}, fmt.Errorf("interval ordering mismatch at %d", index)
	}
	rate, err := e.tariffs.At(index)
	if err != nil {
		return model.SettlementInterval{}, err
	}
	importCost, err := model.CostMicro(site.ImportWh, rate.ImportPriceMicroPerKWh)
	if err != nil {
		return model.SettlementInterval{}, fmt.Errorf("interval %d import cost: %w", index, err)
	}
	exportRevenue, err := model.CostMicro(site.ExportWh, rate.ExportPriceMicroPerKWh)
	if err != nil {
		return model.SettlementInterval{}, fmt.Errorf("interval %d export revenue: %w", index, err)
	}
	carbon, err := model.CarbonGrams(site.ImportWh, rate.CarbonGramsPerKWh)
	if err != nil {
		return model.SettlementInterval{}, fmt.Errorf("interval %d carbon: %w", index, err)
	}
	baselineImportWh := int64(0)
	baselineExportWh := int64(0)
	if baseline.NetSiteW >= 0 {
		baselineImportWh, err = model.PowerToEnergyFloor(baseline.NetSiteW, e.scenario.IntervalMinutes)
	} else {
		baselineExportWh, err = model.PowerToEnergyFloor(-baseline.NetSiteW, e.scenario.IntervalMinutes)
	}
	if err != nil {
		return model.SettlementInterval{}, err
	}
	baselineImportCost, err := model.CostMicro(baselineImportWh, rate.ImportPriceMicroPerKWh)
	if err != nil {
		return model.SettlementInterval{}, err
	}
	baselineExportRevenue, err := model.CostMicro(baselineExportWh, rate.ExportPriceMicroPerKWh)
	if err != nil {
		return model.SettlementInterval{}, err
	}
	baselineCarbon, err := model.CarbonGrams(baselineImportWh, rate.CarbonGramsPerKWh)
	if err != nil {
		return model.SettlementInterval{}, err
	}
	return model.SettlementInterval{
		Index:               index,
		ImportWh:            site.ImportWh,
		ExportWh:            site.ExportWh,
		ImportCostMicro:     importCost,
		ExportRevenueMicro:  exportRevenue,
		NetCostMicro:        importCost - exportRevenue,
		CarbonGrams:         carbon,
		BaselineCostMicro:   baselineImportCost - baselineExportRevenue,
		BaselineCarbonGrams: baselineCarbon,
	}, nil
}

func accumulateInterval(result *model.Settlement, interval model.SettlementInterval) {
	result.TotalImportWh += interval.ImportWh
	result.TotalExportWh += interval.ExportWh
	result.TotalImportCostMicro += interval.ImportCostMicro
	result.TotalExportRevenueMicro += interval.ExportRevenueMicro
	result.CarbonGrams += interval.CarbonGrams
	result.BaselineCostMicro += interval.BaselineCostMicro
	result.BaselineCarbonGrams += interval.BaselineCarbonGrams
}
