package dispatch

import (
	"reflect"
	"testing"
	"time"

	"gridflex/internal/constraint"
	"gridflex/internal/model"
)

func testScenario() model.Scenario {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return model.Scenario{
		ScenarioID:      "dispatch-test",
		Start:           start,
		IntervalMinutes: 15,
		IntervalCount:   4,
		Site:            model.Site{ImportLimitW: 10_000, ExportLimitW: 0},
		Tariffs:         []model.TariffPeriod{{StartIndex: 0, EndIndex: 4, ImportPriceMicroPerKWh: 100_000, CarbonGramsPerKWh: 300}},
		SiteForecast: []model.ForecastPoint{
			{Index: 0, BaseLoadW: 1_000},
			{Index: 1, BaseLoadW: 1_000},
			{Index: 2, BaseLoadW: 1_000},
			{Index: 3, BaseLoadW: 1_000},
		},
		Vehicles: []model.Vehicle{{
			ID: "ev", CapacityWh: 10_000, InitialSOCWh: 1_000,
			MinDepartureSOCWh: 2_000, MinSOCWh: 500, MaxSOCWh: 9_000,
			MaxChargeW: 4_000, ChargeEfficiencyPPM: 1_000_000,
			DischargeEfficiencyPPM: 1_000_000, ArrivalIndex: 0,
			DepartureIndex: 4, FairnessWeight: 1,
		}},
		Options: model.Options{FairnessToleranceWh: 100},
	}
}

func TestPlanIsDeterministicAndFeasible(t *testing.T) {
	scenario := testScenario()
	planner, err := New(scenario)
	if err != nil {
		t.Fatal(err)
	}
	first, err := planner.Plan()
	if err != nil {
		t.Fatal(err)
	}
	secondPlanner, err := New(scenario)
	if err != nil {
		t.Fatal(err)
	}
	second, err := secondPlanner.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("identical input produced different plans")
	}
	if !first.Feasible {
		t.Fatalf("plan infeasible: %v", first.Warnings)
	}
	if violations := constraint.ValidatePlan(scenario, first); len(violations) != 0 {
		t.Fatalf("violations: %#v", violations)
	}
	if got := first.Vehicles[0].FinalSOCWh; got != 2_000 {
		t.Fatalf("final SOC=%d, want 2000", got)
	}
}

func TestCapacityEventLimitsDispatchAndConstraintValidation(t *testing.T) {
	scenario := testScenario()
	scenario.Events = []model.SiteCapacityEvent{{
		ID: "import-curtailment", StartIndex: 0, EndIndex: 1,
		ImportLimitW: 1_000, ExportLimitW: 0,
	}}
	planner, err := New(scenario)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planner.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Site[0].GridPowerW; got != 1_000 {
		t.Fatalf("event interval grid power=%d, want 1000", got)
	}
	for _, action := range plan.Actions {
		if action.IntervalIndex == 0 {
			t.Fatalf("capacity event should prevent interval-0 charging: %#v", action)
		}
	}

	tampered := plan
	tampered.Actions = append(append([]model.Action(nil), plan.Actions...), model.Action{
		IntervalIndex: 0, VehicleID: "ev", PowerW: 1,
		SOCBeforeWh: 1_000, SOCAfterWh: 1_000,
	})
	violations := constraint.ValidatePlan(scenario, tampered)
	found := false
	for _, violation := range violations {
		if violation.Code == "site_import" && violation.IntervalIndex == 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected event-adjusted site_import violation, got %#v", violations)
	}
}
