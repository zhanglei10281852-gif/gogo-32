package settlement

import (
	internal "gridflex/internal/settlement"
	"gridflex/model"
)

type Engine = internal.Engine

func New(scenario model.Scenario) (*Engine, error) {
	return internal.New(scenario)
}

func SummarizeByVehicle(value model.Settlement) map[string]model.VehicleSettlement {
	return internal.SummarizeByVehicle(value)
}

func Reconcile(value model.Settlement) error {
	return internal.Reconcile(value)
}
