package tariff

import (
	internal "gridflex/internal/tariff"
	"gridflex/model"
)

type Rate = internal.Rate
type Schedule = internal.Schedule
type Statistics = internal.Statistics

func New(periods []model.TariffPeriod, count int) (Schedule, error) {
	return internal.New(periods, count)
}

func FromScenario(scenario model.Scenario) (Schedule, error) {
	return internal.FromScenario(scenario)
}
