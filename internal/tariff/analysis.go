package tariff

import (
	"errors"
	"sort"

	"gridflex/internal/model"
)

type Statistics struct {
	MinimumImportPrice int64
	MaximumImportPrice int64
	AverageImportPrice int64
	MinimumExportPrice int64
	MaximumExportPrice int64
	AverageExportPrice int64
	MinimumCarbon      int64
	MaximumCarbon      int64
	AverageCarbon      int64
	ImportSpread       int64
	ExportSpread       int64
}

func (s Schedule) Statistics() Statistics {
	if len(s.rates) == 0 {
		return Statistics{}
	}
	result := Statistics{
		MinimumImportPrice: s.rates[0].ImportPriceMicroPerKWh,
		MaximumImportPrice: s.rates[0].ImportPriceMicroPerKWh,
		MinimumExportPrice: s.rates[0].ExportPriceMicroPerKWh,
		MaximumExportPrice: s.rates[0].ExportPriceMicroPerKWh,
		MinimumCarbon:      s.rates[0].CarbonGramsPerKWh,
		MaximumCarbon:      s.rates[0].CarbonGramsPerKWh,
	}
	for _, rate := range s.rates {
		if rate.ImportPriceMicroPerKWh < result.MinimumImportPrice {
			result.MinimumImportPrice = rate.ImportPriceMicroPerKWh
		}
		if rate.ImportPriceMicroPerKWh > result.MaximumImportPrice {
			result.MaximumImportPrice = rate.ImportPriceMicroPerKWh
		}
		if rate.ExportPriceMicroPerKWh < result.MinimumExportPrice {
			result.MinimumExportPrice = rate.ExportPriceMicroPerKWh
		}
		if rate.ExportPriceMicroPerKWh > result.MaximumExportPrice {
			result.MaximumExportPrice = rate.ExportPriceMicroPerKWh
		}
		if rate.CarbonGramsPerKWh < result.MinimumCarbon {
			result.MinimumCarbon = rate.CarbonGramsPerKWh
		}
		if rate.CarbonGramsPerKWh > result.MaximumCarbon {
			result.MaximumCarbon = rate.CarbonGramsPerKWh
		}
	}
	result.AverageImportPrice = meanRate(s.rates, func(r Rate) int64 { return r.ImportPriceMicroPerKWh })
	result.AverageExportPrice = meanRate(s.rates, func(r Rate) int64 { return r.ExportPriceMicroPerKWh })
	result.AverageCarbon = meanRate(s.rates, func(r Rate) int64 { return r.CarbonGramsPerKWh })
	result.ImportSpread = result.MaximumImportPrice - result.MinimumImportPrice
	result.ExportSpread = result.MaximumExportPrice - result.MinimumExportPrice
	return result
}

func (s Schedule) CheapestWindow(start, end, length int) ([]int, error) {
	if start < 0 || end > len(s.rates) || start >= end {
		return nil, errors.New("invalid tariff window range")
	}
	if length <= 0 || length > end-start {
		return nil, errors.New("invalid tariff window length")
	}
	bestStart := start
	bestScore := int64(0)
	for index := start; index < start+length; index++ {
		bestScore += s.rates[index].ImportPriceMicroPerKWh
	}
	currentScore := bestScore
	for candidate := start + 1; candidate+length <= end; candidate++ {
		currentScore -= s.rates[candidate-1].ImportPriceMicroPerKWh
		currentScore += s.rates[candidate+length-1].ImportPriceMicroPerKWh
		if currentScore < bestScore {
			bestScore = currentScore
			bestStart = candidate
		}
	}
	result := make([]int, length)
	for offset := 0; offset < length; offset++ {
		result[offset] = bestStart + offset
	}
	return result, nil
}

func (s Schedule) LowCarbonOrder(indices []int) ([]int, error) {
	result := append([]int(nil), indices...)
	for _, index := range result {
		if index < 0 || index >= len(s.rates) {
			return nil, errors.New("tariff index outside schedule")
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := s.rates[result[i]]
		right := s.rates[result[j]]
		if left.CarbonGramsPerKWh != right.CarbonGramsPerKWh {
			return left.CarbonGramsPerKWh < right.CarbonGramsPerKWh
		}
		if left.ImportPriceMicroPerKWh != right.ImportPriceMicroPerKWh {
			return left.ImportPriceMicroPerKWh < right.ImportPriceMicroPerKWh
		}
		return left.Index < right.Index
	})
	return result, nil
}

func (s Schedule) ImportCostForSeries(energyWh []int64) (int64, error) {
	if len(energyWh) != len(s.rates) {
		return 0, errors.New("energy series length does not match tariff schedule")
	}
	var total int64
	for index, energy := range energyWh {
		if energy < 0 {
			return 0, errors.New("import energy must be non-negative")
		}
		cost, err := model.CostMicro(energy, s.rates[index].ImportPriceMicroPerKWh)
		if err != nil {
			return 0, err
		}
		total += cost
	}
	return total, nil
}
