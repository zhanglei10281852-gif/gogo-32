package forecast

import (
	"errors"
	"sort"

	"gridflex/internal/model"
)

type Statistics struct {
	PeakBaseLoadW          int64
	PeakRenewableW         int64
	PeakNetLoadW           int64
	MinimumNetLoadW        int64
	AverageBaseLoadW       int64
	AverageRenewableW      int64
	AverageNetLoadW        int64
	RenewableCurtailmentWh int64
	ImportCapacityBreachWh int64
}

func (s Series) Statistics(minutes int64, importLimitW, exportLimitW int64) (Statistics, error) {
	if len(s.points) == 0 {
		return Statistics{}, errors.New("forecast series is empty")
	}
	if minutes <= 0 {
		return Statistics{}, errors.New("interval minutes must be positive")
	}
	result := Statistics{
		PeakBaseLoadW:   s.points[0].BaseLoadW,
		PeakRenewableW:  s.points[0].RenewableW,
		PeakNetLoadW:    s.points[0].NetLoadW,
		MinimumNetLoadW: s.points[0].NetLoadW,
	}
	var baseTotal int64
	var renewableTotal int64
	var netTotal int64
	for _, point := range s.points {
		if point.BaseLoadW > result.PeakBaseLoadW {
			result.PeakBaseLoadW = point.BaseLoadW
		}
		if point.RenewableW > result.PeakRenewableW {
			result.PeakRenewableW = point.RenewableW
		}
		if point.NetLoadW > result.PeakNetLoadW {
			result.PeakNetLoadW = point.NetLoadW
		}
		if point.NetLoadW < result.MinimumNetLoadW {
			result.MinimumNetLoadW = point.NetLoadW
		}
		baseTotal += point.BaseLoadW
		renewableTotal += point.RenewableW
		netTotal += point.NetLoadW
		if point.NetLoadW < -exportLimitW {
			excessW := -exportLimitW - point.NetLoadW
			excessWh, err := model.PowerToEnergyFloor(excessW, minutes)
			if err != nil {
				return Statistics{}, err
			}
			result.RenewableCurtailmentWh += excessWh
		}
		if point.NetLoadW > importLimitW {
			excessW := point.NetLoadW - importLimitW
			excessWh, err := model.PowerToEnergyFloor(excessW, minutes)
			if err != nil {
				return Statistics{}, err
			}
			result.ImportCapacityBreachWh += excessWh
		}
	}
	count := int64(len(s.points))
	result.AverageBaseLoadW = baseTotal / count
	result.AverageRenewableW = renewableTotal / count
	result.AverageNetLoadW = netTotal / count
	return result, nil
}

func (s Series) SurplusIntervals() []int {
	result := make([]int, 0)
	for _, point := range s.points {
		if point.NetLoadW < 0 {
			result = append(result, point.Index)
		}
	}
	return result
}

func (s Series) ImportBreachIntervals(limitW int64) []int {
	result := make([]int, 0)
	for _, point := range s.points {
		if point.NetLoadW > limitW {
			result = append(result, point.Index)
		}
	}
	return result
}

func (s Series) FlattestOrder(indices []int, targetW int64) ([]int, error) {
	result := append([]int(nil), indices...)
	for _, index := range result {
		if index < 0 || index >= len(s.points) {
			return nil, errors.New("forecast index outside series")
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := model.Abs64(s.points[result[i]].NetLoadW - targetW)
		right := model.Abs64(s.points[result[j]].NetLoadW - targetW)
		if left != right {
			return left < right
		}
		return result[i] < result[j]
	})
	return result, nil
}

func (s Series) RampSeries() []int64 {
	if len(s.points) < 2 {
		return nil
	}
	result := make([]int64, len(s.points)-1)
	for index := 1; index < len(s.points); index++ {
		result[index-1] = s.points[index].NetLoadW - s.points[index-1].NetLoadW
	}
	return result
}

func (s Series) MaximumAbsoluteRampW() int64 {
	var maximum int64
	for _, ramp := range s.RampSeries() {
		absolute := model.Abs64(ramp)
		if absolute > maximum {
			maximum = absolute
		}
	}
	return maximum
}
