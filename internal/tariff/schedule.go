package tariff

import (
	"errors"
	"fmt"
	"sort"

	"gridflex/internal/model"
)

type Rate struct {
	Index                  int
	ImportPriceMicroPerKWh int64
	ExportPriceMicroPerKWh int64
	CarbonGramsPerKWh      int64
}

type Schedule struct {
	rates []Rate
}

func New(periods []model.TariffPeriod, count int) (Schedule, error) {
	if count <= 0 {
		return Schedule{}, errors.New("tariff interval count must be positive")
	}
	rates := make([]Rate, count)
	assigned := make([]bool, count)
	ordered := append([]model.TariffPeriod(nil), periods...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].StartIndex != ordered[j].StartIndex {
			return ordered[i].StartIndex < ordered[j].StartIndex
		}
		return ordered[i].EndIndex < ordered[j].EndIndex
	})
	for _, period := range ordered {
		if period.StartIndex < 0 || period.EndIndex > count || period.StartIndex >= period.EndIndex {
			return Schedule{}, fmt.Errorf("invalid tariff range [%d,%d)", period.StartIndex, period.EndIndex)
		}
		if period.ImportPriceMicroPerKWh < 0 || period.ExportPriceMicroPerKWh < 0 || period.CarbonGramsPerKWh < 0 {
			return Schedule{}, errors.New("tariff values must be non-negative")
		}
		for index := period.StartIndex; index < period.EndIndex; index++ {
			if assigned[index] {
				return Schedule{}, fmt.Errorf("tariff overlap at interval %d", index)
			}
			assigned[index] = true
			rates[index] = Rate{
				Index:                  index,
				ImportPriceMicroPerKWh: period.ImportPriceMicroPerKWh,
				ExportPriceMicroPerKWh: period.ExportPriceMicroPerKWh,
				CarbonGramsPerKWh:      period.CarbonGramsPerKWh,
			}
		}
	}
	for index, ok := range assigned {
		if !ok {
			return Schedule{}, fmt.Errorf("tariff missing interval %d", index)
		}
	}
	return Schedule{rates: rates}, nil
}

func FromScenario(scenario model.Scenario) (Schedule, error) {
	return New(scenario.Tariffs, scenario.IntervalCount)
}

func (s Schedule) Count() int {
	return len(s.rates)
}

func (s Schedule) At(index int) (Rate, error) {
	if index < 0 || index >= len(s.rates) {
		return Rate{}, fmt.Errorf("tariff index %d outside schedule", index)
	}
	return s.rates[index], nil
}

func (s Schedule) Rates() []Rate {
	result := make([]Rate, len(s.rates))
	copy(result, s.rates)
	return result
}

func (s Schedule) ImportOrder(indices []int) ([]int, error) {
	result := append([]int(nil), indices...)
	for _, index := range result {
		if index < 0 || index >= len(s.rates) {
			return nil, fmt.Errorf("tariff index %d outside schedule", index)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := s.rates[result[i]]
		right := s.rates[result[j]]
		if left.ImportPriceMicroPerKWh != right.ImportPriceMicroPerKWh {
			return left.ImportPriceMicroPerKWh < right.ImportPriceMicroPerKWh
		}
		if left.CarbonGramsPerKWh != right.CarbonGramsPerKWh {
			return left.CarbonGramsPerKWh < right.CarbonGramsPerKWh
		}
		return left.Index < right.Index
	})
	return result, nil
}

func (s Schedule) ExportOrder(indices []int) ([]int, error) {
	result := append([]int(nil), indices...)
	for _, index := range result {
		if index < 0 || index >= len(s.rates) {
			return nil, fmt.Errorf("tariff index %d outside schedule", index)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := s.rates[result[i]]
		right := s.rates[result[j]]
		if left.ExportPriceMicroPerKWh != right.ExportPriceMicroPerKWh {
			return left.ExportPriceMicroPerKWh > right.ExportPriceMicroPerKWh
		}
		if left.CarbonGramsPerKWh != right.CarbonGramsPerKWh {
			return left.CarbonGramsPerKWh > right.CarbonGramsPerKWh
		}
		return left.Index < right.Index
	})
	return result, nil
}

func (s Schedule) AverageImportPrice() int64 {
	if len(s.rates) == 0 {
		return 0
	}
	var total int64
	for _, rate := range s.rates {
		total += rate.ImportPriceMicroPerKWh
	}
	return total / int64(len(s.rates))
}

func (s Schedule) AverageCarbon() int64 {
	if len(s.rates) == 0 {
		return 0
	}
	var total int64
	for _, rate := range s.rates {
		total += rate.CarbonGramsPerKWh
	}
	return total / int64(len(s.rates))
}

func (s Schedule) IsExportAttractive(index int) (bool, error) {
	rate, err := s.At(index)
	if err != nil {
		return false, err
	}
	return rate.ExportPriceMicroPerKWh > s.AverageImportPrice(), nil
}
