package constraint

import (
	internal "gridflex/internal/constraint"
	"gridflex/model"
)

type Violation = internal.Violation
type CapacityDiagnostic = internal.CapacityDiagnostic
type VehicleDiagnostic = internal.VehicleDiagnostic

func ValidatePlan(scenario model.Scenario, plan model.Plan) []Violation {
	return internal.ValidatePlan(scenario, plan)
}

func IsFeasible(scenario model.Scenario, plan model.Plan) bool {
	return internal.IsFeasible(scenario, plan)
}

func CapacityDiagnostics(scenario model.Scenario, plan model.Plan) ([]CapacityDiagnostic, error) {
	return internal.CapacityDiagnostics(scenario, plan)
}

func VehicleDiagnostics(scenario model.Scenario, plan model.Plan) ([]VehicleDiagnostic, error) {
	return internal.VehicleDiagnostics(scenario, plan)
}

func ViolationCounts(violations []Violation) map[string]int {
	return internal.ViolationCounts(violations)
}

func FirstViolation(violations []Violation) (Violation, bool) {
	return internal.FirstViolation(violations)
}
