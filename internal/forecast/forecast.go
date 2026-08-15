package forecast

import (
	"errors"
	"fmt"
	"sort"

	"gridflex/internal/model"
)

type Point struct {
	Index      int
	BaseLoadW  int64
	RenewableW int64
	NetLoadW   int64
}

type Series struct {
	points []Point
}

func New(input []model.ForecastPoint, count int) (Series, error) {
	if count <= 0 {
		return Series{}, errors.New("forecast count must be positive")
	}
	if len(input) != count {
		return Series{}, fmt.Errorf("forecast length %d does not match count %d", len(input), count)
	}
	ordered := append([]model.ForecastPoint(nil), input...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Index < ordered[j].Index
	})
	points := make([]Point, count)
	for expected, item := range ordered {
		if item.Index != expected {
			return Series{}, fmt.Errorf("forecast index %d, expected %d", item.Index, expected)
		}
		if item.BaseLoadW < 0 || item.RenewableW < 0 {
			return Series{}, fmt.Errorf("forecast interval %d has negative value", item.Index)
		}
		points[expected] = Point{
			Index:      expected,
			BaseLoadW:  item.BaseLoadW,
			RenewableW: item.RenewableW,
			NetLoadW:   item.BaseLoadW - item.RenewableW,
		}
	}
	return Series{points: points}, nil
}

func FromScenario(scenario model.Scenario) (Series, error) {
	return New(scenario.SiteForecast, scenario.IntervalCount)
}

func (s Series) Count() int {
	return len(s.points)
}

func (s Series) At(index int) (Point, error) {
	if index < 0 || index >= len(s.points) {
		return Point{}, fmt.Errorf("forecast index %d outside series", index)
	}
	return s.points[index], nil
}

func (s Series) Points() []Point {
	result := make([]Point, len(s.points))
	copy(result, s.points)
	return result
}

func (s Series) PeakNetLoadW() int64 {
	var peak int64
	for index, point := range s.points {
		if index == 0 || point.NetLoadW > peak {
			peak = point.NetLoadW
		}
	}
	return peak
}

func (s Series) MinimumNetLoadW() int64 {
	var minimum int64
	for index, point := range s.points {
		if index == 0 || point.NetLoadW < minimum {
			minimum = point.NetLoadW
		}
	}
	return minimum
}

func (s Series) TotalBaseEnergyWh(minutes int64) (int64, error) {
	if minutes <= 0 {
		return 0, errors.New("interval minutes must be positive")
	}
	var total int64
	for _, point := range s.points {
		energy, err := model.PowerToEnergyFloor(point.BaseLoadW, minutes)
		if err != nil {
			return 0, err
		}
		total += energy
	}
	return total, nil
}

func (s Series) TotalRenewableEnergyWh(minutes int64) (int64, error) {
	if minutes <= 0 {
		return 0, errors.New("interval minutes must be positive")
	}
	var total int64
	for _, point := range s.points {
		energy, err := model.PowerToEnergyFloor(point.RenewableW, minutes)
		if err != nil {
			return 0, err
		}
		total += energy
	}
	return total, nil
}

func (s Series) HeadroomW(index int, importLimitW int64) (int64, error) {
	point, err := s.At(index)
	if err != nil {
		return 0, err
	}
	headroom := importLimitW - point.NetLoadW
	if headroom < 0 {
		return 0, nil
	}
	return headroom, nil
}

func (s Series) ExportHeadroomW(index int, exportLimitW int64) (int64, error) {
	point, err := s.At(index)
	if err != nil {
		return 0, err
	}
	headroom := exportLimitW + point.NetLoadW
	if headroom < 0 {
		return 0, nil
	}
	return headroom, nil
}

func (s Series) NetLoadOrder(indices []int) ([]int, error) {
	result := append([]int(nil), indices...)
	for _, index := range result {
		if index < 0 || index >= len(s.points) {
			return nil, fmt.Errorf("forecast index %d outside series", index)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := s.points[result[i]]
		right := s.points[result[j]]
		if left.NetLoadW != right.NetLoadW {
			return left.NetLoadW < right.NetLoadW
		}
		return left.Index < right.Index
	})
	return result, nil
}
