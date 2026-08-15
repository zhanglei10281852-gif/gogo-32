package store

import (
	"testing"
	"time"

	"gridflex/internal/model"
)

func auditScenario() model.Scenario {
	return model.Scenario{
		ScenarioID:      "audit",
		Start:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		IntervalMinutes: 15,
		IntervalCount:   1,
		Site:            model.Site{ImportLimitW: 10_000, ExportLimitW: 0},
		Events: []model.SiteCapacityEvent{{
			ID: "curtailment", StartIndex: 0, EndIndex: 1,
			ImportLimitW: 5_000, ExportLimitW: 0,
		}},
		Tariffs:      []model.TariffPeriod{{StartIndex: 0, EndIndex: 1, ImportPriceMicroPerKWh: 100_000, CarbonGramsPerKWh: 300}},
		SiteForecast: []model.ForecastPoint{{Index: 0, BaseLoadW: 1_000}},
		Vehicles: []model.Vehicle{{
			ID: "ev", CapacityWh: 10_000, InitialSOCWh: 1_000,
			MinDepartureSOCWh: 1_000, MinSOCWh: 500, MaxSOCWh: 9_000,
			MaxChargeW: 4_000, ChargeEfficiencyPPM: 1_000_000,
			DischargeEfficiencyPPM: 1_000_000, ArrivalIndex: 0,
			DepartureIndex: 1, FairnessWeight: 1,
		}},
		Options: model.Options{FairnessToleranceWh: 100},
	}
}

func newAuditedStore(t *testing.T) (*Store, model.AuditSnapshot) {
	t.Helper()
	storage, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scenario := auditScenario()
	if err := storage.SaveInput(scenario); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveBaseline(model.Baseline{ScenarioID: scenario.ScenarioID, CreatedAt: scenario.Start}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SavePlan(model.Plan{ScenarioID: scenario.ScenarioID, CreatedAt: scenario.Start, Feasible: true}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveSettlement(model.Settlement{ScenarioID: scenario.ScenarioID, CreatedAt: scenario.Start}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := storage.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return storage, snapshot
}

func TestAuditFieldsArePresentStableAndVerified(t *testing.T) {
	storage, first := newAuditedStore(t)
	if first.InputSHA256 == "" || first.ResultSHA256 == "" || first.RootSHA256 == "" {
		t.Fatalf("required hashes are missing: %#v", first)
	}
	if first.AlgorithmVersion != AlgorithmVersion || first.Config.ScenarioID != "audit" || len(first.Config.Events) != 1 {
		t.Fatalf("required config/version fields are incomplete: %#v", first)
	}
	if len(first.Entries) != 4 {
		t.Fatalf("audit entries=%d, want 4", len(first.Entries))
	}
	if err := storage.VerifyAudit(); err != nil {
		t.Fatal(err)
	}
	second, err := storage.CreateAudit()
	if err != nil {
		t.Fatal(err)
	}
	if first.RootSHA256 != second.RootSHA256 || first.InputSHA256 != second.InputSHA256 || first.ResultSHA256 != second.ResultSHA256 {
		t.Fatal("stable artifacts produced unstable audit hashes")
	}
}

func TestAuditDetectsInputAndResultMutation(t *testing.T) {
	t.Run("input", func(t *testing.T) {
		storage, _ := newAuditedStore(t)
		scenario, err := storage.LoadInput()
		if err != nil {
			t.Fatal(err)
		}
		scenario.Site.ImportLimitW--
		if err := storage.SaveInput(scenario); err != nil {
			t.Fatal(err)
		}
		if err := storage.VerifyAudit(); err == nil {
			t.Fatal("expected input mutation to fail audit verification")
		}
	})

	t.Run("result", func(t *testing.T) {
		storage, _ := newAuditedStore(t)
		plan, err := storage.LoadPlan()
		if err != nil {
			t.Fatal(err)
		}
		plan.Feasible = false
		if err := storage.SavePlan(plan); err != nil {
			t.Fatal(err)
		}
		if err := storage.VerifyAudit(); err == nil {
			t.Fatal("expected result mutation to fail audit verification")
		}
	})
}

func TestArtifactLifecycleIncludesInput(t *testing.T) {
	storage, _ := newAuditedStore(t)
	complete, err := storage.Complete()
	if err != nil || !complete {
		t.Fatalf("complete=%t err=%v", complete, err)
	}
	catalog, err := storage.Catalog()
	if err != nil || len(catalog) != 5 {
		t.Fatalf("catalog length=%d err=%v", len(catalog), err)
	}
	if _, err := storage.ArtifactPath(InputFile); err != nil {
		t.Fatal(err)
	}
	if err := storage.RemoveArtifacts(); err != nil {
		t.Fatal(err)
	}
	catalog, err = storage.Catalog()
	if err != nil || len(catalog) != 0 {
		t.Fatalf("catalog after remove=%d err=%v", len(catalog), err)
	}
}
