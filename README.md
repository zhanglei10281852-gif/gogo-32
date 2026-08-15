# GridFlex

GridFlex is an offline, deterministic fleet charging/discharging planner written with the Go standard library. It validates a JSON scenario, builds a time-grid baseline and constrained plan, settles energy/cost/revenue/carbon, persists immutable JSON artifacts, and emits text or JSON reports.

This independent software is not affiliated with, endorsed by, sponsored by, or otherwise connected to the World Economic Forum. Its Everything-to-grid theme was inspired by the technology overview at https://www.weforum.org/stories/emerging-technologies/the-top-10-emerging-technologies-of-2026/ . The link is provided only as news context and does not imply affiliation or endorsement.

## Build and test

```powershell
go test ./...
go build ./...
go vet ./...
```

## CLI

```powershell
go run ./cmd/gridflex validate -input scenario.json
go run ./cmd/gridflex plan -input scenario.json -out artifacts
go run ./cmd/gridflex report -dir artifacts -format text
go run ./cmd/gridflex report -dir artifacts -format json
go run ./cmd/gridflex run -input scenario.json -out artifacts -format text
```

All commands are offline. `validate` performs strict JSON decoding (unknown fields and trailing values are rejected). `plan` writes the normalized `input.json` plus `baseline.json`, `plan.json`, `settlement.json`, and `audit.json`. `report` verifies every persisted input/result and all derived audit hashes before rendering. `run` performs validate, plan, persist, verify, and report.

## Input example

Times are RFC3339 and interval boundaries are half-open. Energy is Wh, power W, prices micro-currency/kWh, carbon g/kWh, and efficiencies are ppm.

```json
{
  "scenario_id": "demo-001",
  "start": "2026-01-01T00:00:00Z",
  "interval_minutes": 15,
  "interval_count": 8,
  "site": { "import_limit_w": 22000, "export_limit_w": 8000 },
  "events": [
    {
      "id": "grid-maintenance",
      "start_index": 2,
      "end_index": 4,
      "import_limit_w": 12000,
      "export_limit_w": 2000
    }
  ],
  "tariffs": [
    {
      "start_index": 0,
      "end_index": 4,
      "import_price_micro_per_kwh": 120000,
      "export_price_micro_per_kwh": 50000,
      "carbon_g_per_kwh": 350
    },
    {
      "start_index": 4,
      "end_index": 8,
      "import_price_micro_per_kwh": 300000,
      "export_price_micro_per_kwh": 180000,
      "carbon_g_per_kwh": 500
    }
  ],
  "site_forecast": [
    { "index": 0, "base_load_w": 6000, "renewable_w": 1000 },
    { "index": 1, "base_load_w": 6200, "renewable_w": 1000 },
    { "index": 2, "base_load_w": 6500, "renewable_w": 1200 },
    { "index": 3, "base_load_w": 7000, "renewable_w": 1500 },
    { "index": 4, "base_load_w": 8000, "renewable_w": 1000 },
    { "index": 5, "base_load_w": 8500, "renewable_w": 800 },
    { "index": 6, "base_load_w": 7500, "renewable_w": 500 },
    { "index": 7, "base_load_w": 7000, "renewable_w": 500 }
  ],
  "vehicles": [
    {
      "id": "ev-a",
      "capacity_wh": 60000,
      "initial_soc_wh": 18000,
      "min_departure_soc_wh": 30000,
      "min_soc_wh": 6000,
      "max_soc_wh": 57000,
      "max_charge_w": 11000,
      "max_discharge_w": 5000,
      "charge_efficiency_ppm": 930000,
      "discharge_efficiency_ppm": 920000,
      "arrival_index": 0,
      "departure_index": 8,
      "allow_discharge": true,
      "fairness_weight": 1
    },
    {
      "id": "ev-b",
      "capacity_wh": 45000,
      "initial_soc_wh": 12000,
      "min_departure_soc_wh": 24000,
      "min_soc_wh": 4500,
      "max_soc_wh": 43000,
      "max_charge_w": 7000,
      "max_discharge_w": 3000,
      "charge_efficiency_ppm": 950000,
      "discharge_efficiency_ppm": 900000,
      "arrival_index": 1,
      "departure_index": 7,
      "allow_discharge": false,
      "fairness_weight": 2
    }
  ],
  "options": {
    "enable_export": true,
    "fairness_tolerance_wh": 1000,
    "reserve_margin_wh": 500
  }
}
```

## Capacity events and audit integrity

`events` is optional. Each event has a unique non-empty `id` and a half-open interval `[start_index, end_index)`. During that interval its `import_limit_w` and `export_limit_w` replace both default `site` limits. Events are normalized by start index, end index, then ID; overlaps are rejected, while adjacent intervals are allowed. IDs, bounds, limits, duplicates, and unknown JSON fields are validated strictly. Dispatch, post-plan constraint validation, and capacity diagnostics all use the same effective limits.

`audit.json` retains per-file `entries` and `root_sha256` and also records `input_sha256`, the complete normalized scenario under `config`, `algorithm_version` (`gridflex-dispatch-v1`), and `result_sha256`. The input hash is computed from compact stable JSON for the normalized scenario. The result hash is a length-framed deterministic combination of canonical baseline, plan, and settlement JSON. `input.json` is also a raw-file audit entry, so both semantic changes and formatting/file changes are detected. Verification reloads and validates the normalized input, recomputes every field, checks scenario IDs, and rejects any mismatch.

## Determinism and units

Core arithmetic uses signed 64-bit integers. Vehicle IDs and interval indices define stable tie-breaking. Rational conversions use explicit floor/ceiling helpers, so a repeated run on identical input produces byte-stable JSON and the same SHA-256 audit hash. The planner enforces availability, charge/discharge power, SOC bounds, efficiencies, departure targets, site import/export limits, and weighted fairness.
