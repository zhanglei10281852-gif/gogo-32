package dispatch_test

import (
	"math"
	"testing"
	"time"

	"gridflex/dispatch"
	"gridflex/model"
)

func TestHugeImportPricesDoNotMakeCheapExportAttractive(t *testing.T) {
	scenario := model.Scenario{
		ScenarioID: "large-tariff", Start: time.Unix(0, 0).UTC(),
		IntervalMinutes: 60, IntervalCount: 2,
		Site: model.Site{ImportLimitW: 1000, ExportLimitW: 1000},
		Tariffs: []model.TariffPeriod{
			{StartIndex: 0, EndIndex: 1, ImportPriceMicroPerKWh: math.MaxInt64, ExportPriceMicroPerKWh: 1},
			{StartIndex: 1, EndIndex: 2, ImportPriceMicroPerKWh: math.MaxInt64, ExportPriceMicroPerKWh: 1},
		},
		SiteForecast: []model.ForecastPoint{{Index: 0}, {Index: 1}},
		Vehicles: []model.Vehicle{{
			ID: "ev", CapacityWh: 1000, InitialSOCWh: 900,
			MinDepartureSOCWh: 500, MinSOCWh: 100, MaxSOCWh: 1000,
			MaxChargeW: 100, MaxDischargeW: 100, AllowDischarge: true,
			ChargeEfficiencyPPM:    model.PartsPerMillion,
			DischargeEfficiencyPPM: model.PartsPerMillion,
			ArrivalIndex:           0, DepartureIndex: 2, FairnessWeight: 1,
		}},
		Options: model.Options{EnableExport: true},
	}
	planner, err := dispatch.New(scenario)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planner.Plan()
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range plan.Actions {
		if action.PowerW < 0 {
			t.Fatalf("unexpected V2G action for negligible export price: %#v", action)
		}
	}
}
