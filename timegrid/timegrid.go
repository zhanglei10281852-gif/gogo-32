package timegrid

import (
	"time"

	internal "gridflex/internal/timegrid"
	"gridflex/model"
)

type Grid = internal.Grid
type Window = internal.Window

func New(startTime time.Time, minutes int64, count int) (Grid, error) {
	return internal.New(startTime, minutes, count)
}

func FromScenario(scenario model.Scenario) (Grid, error) {
	return internal.FromScenario(scenario)
}

func MergeWindows(input []Window) []Window {
	return internal.MergeWindows(input)
}

func ComplementWindows(count int, occupied []Window) ([]Window, error) {
	return internal.ComplementWindows(count, occupied)
}
