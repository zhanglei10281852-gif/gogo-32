package dispatch

import (
	internal "gridflex/internal/dispatch"
	"gridflex/model"
)

type Planner = internal.Planner

func New(scenario model.Scenario) (*Planner, error) {
	return internal.New(scenario)
}

func BuildBaseline(scenario model.Scenario) (model.Baseline, error) {
	return internal.BuildBaseline(scenario)
}
