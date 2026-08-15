package timegrid

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"gridflex/internal/model"
)

type Grid struct {
	start     time.Time
	minutes   int64
	intervals []model.Interval
}

func New(start time.Time, minutes int64, count int) (Grid, error) {
	if start.IsZero() {
		return Grid{}, errors.New("grid start is required")
	}
	if minutes <= 0 {
		return Grid{}, errors.New("grid interval minutes must be positive")
	}
	if count <= 0 {
		return Grid{}, errors.New("grid count must be positive")
	}
	if minutes > int64((1<<63-1)/int64(time.Minute)) {
		return Grid{}, errors.New("grid duration overflows")
	}
	duration := time.Duration(minutes) * time.Minute
	intervals := make([]model.Interval, count)
	cursor := start
	for index := 0; index < count; index++ {
		next := cursor.Add(duration)
		if !next.After(cursor) {
			return Grid{}, errors.New("grid time overflow")
		}
		intervals[index] = model.Interval{Index: index, Start: cursor, End: next}
		cursor = next
	}
	return Grid{start: start, minutes: minutes, intervals: intervals}, nil
}

func FromScenario(scenario model.Scenario) (Grid, error) {
	return New(scenario.Start, scenario.IntervalMinutes, scenario.IntervalCount)
}

func (g Grid) Start() time.Time {
	return g.start
}

func (g Grid) Minutes() int64 {
	return g.minutes
}

func (g Grid) Count() int {
	return len(g.intervals)
}

func (g Grid) End() time.Time {
	if len(g.intervals) == 0 {
		return g.start
	}
	return g.intervals[len(g.intervals)-1].End
}

func (g Grid) Interval(index int) (model.Interval, error) {
	if index < 0 || index >= len(g.intervals) {
		return model.Interval{}, fmt.Errorf("interval index %d outside grid", index)
	}
	return g.intervals[index], nil
}

func (g Grid) Intervals() []model.Interval {
	result := make([]model.Interval, len(g.intervals))
	copy(result, g.intervals)
	return result
}

func (g Grid) Contains(index int) bool {
	return index >= 0 && index < len(g.intervals)
}

func (g Grid) Available(arrival, departure, index int) bool {
	return g.Contains(index) && index >= arrival && index < departure
}

func (g Grid) Remaining(arrival, departure, current int) int {
	if current < arrival {
		current = arrival
	}
	if current >= departure {
		return 0
	}
	if departure > len(g.intervals) {
		departure = len(g.intervals)
	}
	return departure - current
}

func (g Grid) IndexAt(value time.Time) (int, error) {
	if value.Before(g.start) || !value.Before(g.End()) {
		return 0, fmt.Errorf("time %s outside grid", value.Format(time.RFC3339Nano))
	}
	delta := value.Sub(g.start)
	duration := time.Duration(g.minutes) * time.Minute
	return int(delta / duration), nil
}

func (g Grid) IsBoundary(value time.Time) bool {
	if value.Before(g.start) || value.After(g.End()) {
		return false
	}
	duration := time.Duration(g.minutes) * time.Minute
	return value.Sub(g.start)%duration == 0
}

func (g Grid) ValidateIndices(indices []int) error {
	copyIndices := append([]int(nil), indices...)
	sort.Ints(copyIndices)
	for position, index := range copyIndices {
		if !g.Contains(index) {
			return fmt.Errorf("index %d outside grid", index)
		}
		if position > 0 && index == copyIndices[position-1] {
			return fmt.Errorf("index %d appears more than once", index)
		}
	}
	return nil
}

func (g Grid) EnergyAtPowerFloor(powerW int64) (int64, error) {
	if powerW < 0 {
		return 0, errors.New("power must be non-negative")
	}
	return model.PowerToEnergyFloor(powerW, g.minutes)
}

func (g Grid) EnergyAtPowerCeil(powerW int64) (int64, error) {
	if powerW < 0 {
		return 0, errors.New("power must be non-negative")
	}
	return model.PowerToEnergyCeil(powerW, g.minutes)
}

func (g Grid) PowerForEnergyFloor(energyWh int64) (int64, error) {
	if energyWh < 0 {
		return 0, errors.New("energy must be non-negative")
	}
	return model.EnergyToPowerFloor(energyWh, g.minutes)
}
