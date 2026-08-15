package forecast

import (
	internal "gridflex/internal/forecast"
	"gridflex/model"
)

type Point = internal.Point
type Series = internal.Series
type Statistics = internal.Statistics

func New(points []model.ForecastPoint, count int) (Series, error) {
	return internal.New(points, count)
}

func FromScenario(scenario model.Scenario) (Series, error) {
	return internal.FromScenario(scenario)
}
