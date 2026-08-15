package config

import (
	"strings"
	"testing"
)

const validScenarioJSON = `{
  "scenario_id":"unit",
  "start":"2026-01-01T00:00:00Z",
  "interval_minutes":15,
  "interval_count":4,
  "site":{"import_limit_w":10000,"export_limit_w":0},
  "events":[
    {"id":" later ","start_index":2,"end_index":3,"import_limit_w":5000,"export_limit_w":0},
    {"id":"early","start_index":0,"end_index":1,"import_limit_w":3000,"export_limit_w":0}
  ],
  "tariffs":[{"start_index":0,"end_index":4,"import_price_micro_per_kwh":100000,"export_price_micro_per_kwh":0,"carbon_g_per_kwh":300}],
  "site_forecast":[
    {"index":0,"base_load_w":1000,"renewable_w":0},
    {"index":1,"base_load_w":1000,"renewable_w":0},
    {"index":2,"base_load_w":1000,"renewable_w":0},
    {"index":3,"base_load_w":1000,"renewable_w":0}
  ],
  "vehicles":[{"id":"ev","capacity_wh":10000,"initial_soc_wh":1000,"min_departure_soc_wh":2000,"min_soc_wh":500,"max_soc_wh":9000,"max_charge_w":4000,"max_discharge_w":0,"charge_efficiency_ppm":1000000,"discharge_efficiency_ppm":1000000,"arrival_index":0,"departure_index":4,"allow_discharge":false,"fairness_weight":1}],
  "options":{"enable_export":false,"fairness_tolerance_wh":100,"reserve_margin_wh":0}
}`

func TestDecodeStrictAndNormalizes(t *testing.T) {
	scenario, err := Decode(strings.NewReader(validScenarioJSON))
	if err != nil {
		t.Fatal(err)
	}
	if scenario.ScenarioID != "unit" || len(scenario.Vehicles) != 1 {
		t.Fatalf("unexpected scenario: %#v", scenario)
	}
	if len(scenario.Events) != 2 || scenario.Events[0].ID != "early" || scenario.Events[1].ID != "later" {
		t.Fatalf("events were not normalized and sorted: %#v", scenario.Events)
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	invalid := strings.Replace(validScenarioJSON, `"scenario_id":"unit"`, `"scenario_id":"unit","mystery":1`, 1)
	if _, err := Decode(strings.NewReader(invalid)); err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestDecodeRejectsTrailingValue(t *testing.T) {
	if _, err := Decode(strings.NewReader(validScenarioJSON + `{}`)); err == nil {
		t.Fatal("expected trailing-value error")
	}
}

func TestDecodeRejectsOverlappingCapacityEvents(t *testing.T) {
	invalid := strings.Replace(validScenarioJSON,
		`{"id":" later ","start_index":2,"end_index":3,"import_limit_w":5000,"export_limit_w":0}`,
		`{"id":" later ","start_index":0,"end_index":3,"import_limit_w":5000,"export_limit_w":0}`,
		1,
	)
	if _, err := Decode(strings.NewReader(invalid)); err == nil || !strings.Contains(err.Error(), "overlaps") {
		t.Fatalf("expected overlap error, got %v", err)
	}
}

func TestDecodeRejectsInvalidEventAndNegativeHorizonWithoutPanic(t *testing.T) {
	invalidEvent := strings.Replace(validScenarioJSON, `"import_limit_w":3000`, `"import_limit_w":0`, 1)
	if _, err := Decode(strings.NewReader(invalidEvent)); err == nil || !strings.Contains(err.Error(), "events[0].import_limit_w") {
		t.Fatalf("expected event capacity error, got %v", err)
	}
	negativeHorizon := strings.Replace(validScenarioJSON, `"interval_count":4`, `"interval_count":-1`, 1)
	if _, err := Decode(strings.NewReader(negativeHorizon)); err == nil || !strings.Contains(err.Error(), "interval_count must be positive") {
		t.Fatalf("expected horizon error, got %v", err)
	}
}
