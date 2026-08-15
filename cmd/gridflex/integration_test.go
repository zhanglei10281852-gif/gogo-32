package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const integrationInput = `{
  "scenario_id":"e2e",
  "start":"2026-01-01T00:00:00Z",
  "interval_minutes":15,
  "interval_count":4,
  "site":{"import_limit_w":10000,"export_limit_w":0},
  "events":[{"id":"brief-curtailment","start_index":0,"end_index":1,"import_limit_w":2000,"export_limit_w":0}],
  "tariffs":[{"start_index":0,"end_index":4,"import_price_micro_per_kwh":100000,"export_price_micro_per_kwh":0,"carbon_g_per_kwh":300}],
  "site_forecast":[{"index":0,"base_load_w":1000,"renewable_w":0},{"index":1,"base_load_w":1000,"renewable_w":0},{"index":2,"base_load_w":1000,"renewable_w":0},{"index":3,"base_load_w":1000,"renewable_w":0}],
  "vehicles":[{"id":"ev","capacity_wh":10000,"initial_soc_wh":1000,"min_departure_soc_wh":2000,"min_soc_wh":500,"max_soc_wh":9000,"max_charge_w":4000,"max_discharge_w":0,"charge_efficiency_ppm":1000000,"discharge_efficiency_ppm":1000000,"arrival_index":0,"departure_index":4,"allow_discharge":false,"fairness_weight":1}],
  "options":{"enable_export":false,"fairness_tolerance_wh":100,"reserve_margin_wh":0}
}`

func TestRunEndToEndAndReportJSON(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "scenario.json")
	artifacts := filepath.Join(directory, "artifacts")
	if err := os.WriteFile(input, []byte(integrationInput), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"run", "-input", input, "-out", artifacts, "-format", "text"}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v, stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "GRIDFLEX REPORT") || !strings.Contains(stdout.String(), "scenario: e2e") {
		t.Fatalf("unexpected text report: %s", stdout.String())
	}
	for _, name := range []string{"input.json", "baseline.json", "plan.json", "settlement.json", "audit.json"} {
		if _, err := os.Stat(filepath.Join(artifacts, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	stdout.Reset()
	if err := run([]string{"report", "-dir", artifacts, "-format", "json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"scenario_id": "e2e"`) ||
		!strings.Contains(stdout.String(), `"input_sha256":`) ||
		!strings.Contains(stdout.String(), `"algorithm_version": "gridflex-dispatch-v1"`) ||
		!strings.Contains(stdout.String(), `"result_sha256":`) ||
		!strings.Contains(stdout.String(), `"config":`) {
		t.Fatalf("unexpected JSON report: %s", stdout.String())
	}
}

func TestValidateCommand(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "scenario.json")
	if err := os.WriteFile(input, []byte(integrationInput), 0o644); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"validate", "-input", input}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "valid scenario e2e") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}
