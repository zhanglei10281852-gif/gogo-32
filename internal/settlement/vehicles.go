package settlement

import (
	"fmt"
	"sort"

	"gridflex/internal/model"
)

type vehicleAccumulator struct {
	importWh           int64
	exportWh           int64
	energyCostMicro    int64
	exportRevenueMicro int64
	carbonGrams        int64
}

func (e *Engine) settleVehicles(actions []model.Action) ([]model.VehicleSettlement, error) {
	accumulators := make(map[string]*vehicleAccumulator, len(e.scenario.Vehicles))
	for _, vehicle := range e.scenario.Vehicles {
		accumulators[vehicle.ID] = &vehicleAccumulator{}
	}
	ordered := append([]model.Action(nil), actions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].IntervalIndex != ordered[j].IntervalIndex {
			return ordered[i].IntervalIndex < ordered[j].IntervalIndex
		}
		return ordered[i].VehicleID < ordered[j].VehicleID
	})
	for _, action := range ordered {
		accumulator, ok := accumulators[action.VehicleID]
		if !ok {
			return nil, fmt.Errorf("settlement action references unknown vehicle %q", action.VehicleID)
		}
		rate, err := e.tariffs.At(action.IntervalIndex)
		if err != nil {
			return nil, err
		}
		if action.PowerW > 0 {
			cost, err := model.CostMicro(action.GridEnergyWh, rate.ImportPriceMicroPerKWh)
			if err != nil {
				return nil, err
			}
			carbon, err := model.CarbonGrams(action.GridEnergyWh, rate.CarbonGramsPerKWh)
			if err != nil {
				return nil, err
			}
			accumulator.importWh += action.GridEnergyWh
			accumulator.energyCostMicro += cost
			accumulator.carbonGrams += carbon
		}
		if action.PowerW < 0 {
			revenue, err := model.CostMicro(action.GridEnergyWh, rate.ExportPriceMicroPerKWh)
			if err != nil {
				return nil, err
			}
			accumulator.exportWh += action.GridEnergyWh
			accumulator.exportRevenueMicro += revenue
		}
	}
	ids := make([]string, 0, len(accumulators))
	for id := range accumulators {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]model.VehicleSettlement, 0, len(ids))
	for _, id := range ids {
		accumulator := accumulators[id]
		result = append(result, model.VehicleSettlement{
			VehicleID:          id,
			ImportWh:           accumulator.importWh,
			ExportWh:           accumulator.exportWh,
			EnergyCostMicro:    accumulator.energyCostMicro,
			ExportRevenueMicro: accumulator.exportRevenueMicro,
			NetCostMicro:       accumulator.energyCostMicro - accumulator.exportRevenueMicro,
			CarbonGrams:        accumulator.carbonGrams,
		})
	}
	return result, nil
}

func SummarizeByVehicle(settlement model.Settlement) map[string]model.VehicleSettlement {
	result := make(map[string]model.VehicleSettlement, len(settlement.Vehicles))
	for _, vehicle := range settlement.Vehicles {
		result[vehicle.VehicleID] = vehicle
	}
	return result
}

func Reconcile(settlement model.Settlement) error {
	var intervalImport int64
	var intervalExport int64
	var intervalCost int64
	var intervalRevenue int64
	var intervalCarbon int64
	for _, interval := range settlement.Intervals {
		intervalImport += interval.ImportWh
		intervalExport += interval.ExportWh
		intervalCost += interval.ImportCostMicro
		intervalRevenue += interval.ExportRevenueMicro
		intervalCarbon += interval.CarbonGrams
	}
	if intervalImport != settlement.TotalImportWh {
		return fmt.Errorf("total import %d does not reconcile with intervals %d", settlement.TotalImportWh, intervalImport)
	}
	if intervalExport != settlement.TotalExportWh {
		return fmt.Errorf("total export %d does not reconcile with intervals %d", settlement.TotalExportWh, intervalExport)
	}
	if intervalCost != settlement.TotalImportCostMicro {
		return fmt.Errorf("total import cost %d does not reconcile with intervals %d", settlement.TotalImportCostMicro, intervalCost)
	}
	if intervalRevenue != settlement.TotalExportRevenueMicro {
		return fmt.Errorf("total export revenue %d does not reconcile with intervals %d", settlement.TotalExportRevenueMicro, intervalRevenue)
	}
	if intervalCarbon != settlement.CarbonGrams {
		return fmt.Errorf("total carbon %d does not reconcile with intervals %d", settlement.CarbonGrams, intervalCarbon)
	}
	if settlement.NetCostMicro != settlement.TotalImportCostMicro-settlement.TotalExportRevenueMicro {
		return fmt.Errorf("net cost does not reconcile")
	}
	if settlement.SavingsMicro != settlement.BaselineCostMicro-settlement.NetCostMicro {
		return fmt.Errorf("savings does not reconcile")
	}
	return nil
}
