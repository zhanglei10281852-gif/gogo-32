package model

import "time"

type Scenario struct {
	ScenarioID      string              `json:"scenario_id"`
	Start           time.Time           `json:"start"`
	IntervalMinutes int64               `json:"interval_minutes"`
	IntervalCount   int                 `json:"interval_count"`
	Site            Site                `json:"site"`
	Events          []SiteCapacityEvent `json:"events"`
	Tariffs         []TariffPeriod      `json:"tariffs"`
	SiteForecast    []ForecastPoint     `json:"site_forecast"`
	Vehicles        []Vehicle           `json:"vehicles"`
	Options         Options             `json:"options"`
}

type Site struct {
	ImportLimitW int64 `json:"import_limit_w"`
	ExportLimitW int64 `json:"export_limit_w"`
}

// SiteCapacityEvent replaces both site capacity limits for the half-open
// interval [StartIndex, EndIndex). Valid scenarios do not contain overlaps.
type SiteCapacityEvent struct {
	ID           string `json:"id"`
	StartIndex   int    `json:"start_index"`
	EndIndex     int    `json:"end_index"`
	ImportLimitW int64  `json:"import_limit_w"`
	ExportLimitW int64  `json:"export_limit_w"`
}

// SiteCapacityAt returns the event-adjusted capacity for an interval.
func SiteCapacityAt(scenario Scenario, index int) Site {
	for _, event := range scenario.Events {
		if index >= event.StartIndex && index < event.EndIndex {
			return Site{ImportLimitW: event.ImportLimitW, ExportLimitW: event.ExportLimitW}
		}
	}
	return scenario.Site
}

type Options struct {
	EnableExport        bool  `json:"enable_export"`
	FairnessToleranceWh int64 `json:"fairness_tolerance_wh"`
	ReserveMarginWh     int64 `json:"reserve_margin_wh"`
}

type TariffPeriod struct {
	StartIndex             int   `json:"start_index"`
	EndIndex               int   `json:"end_index"`
	ImportPriceMicroPerKWh int64 `json:"import_price_micro_per_kwh"`
	ExportPriceMicroPerKWh int64 `json:"export_price_micro_per_kwh"`
	CarbonGramsPerKWh      int64 `json:"carbon_g_per_kwh"`
}

type ForecastPoint struct {
	Index      int   `json:"index"`
	BaseLoadW  int64 `json:"base_load_w"`
	RenewableW int64 `json:"renewable_w"`
}

type Vehicle struct {
	ID                     string `json:"id"`
	CapacityWh             int64  `json:"capacity_wh"`
	InitialSOCWh           int64  `json:"initial_soc_wh"`
	MinDepartureSOCWh      int64  `json:"min_departure_soc_wh"`
	MinSOCWh               int64  `json:"min_soc_wh"`
	MaxSOCWh               int64  `json:"max_soc_wh"`
	MaxChargeW             int64  `json:"max_charge_w"`
	MaxDischargeW          int64  `json:"max_discharge_w"`
	ChargeEfficiencyPPM    int64  `json:"charge_efficiency_ppm"`
	DischargeEfficiencyPPM int64  `json:"discharge_efficiency_ppm"`
	ArrivalIndex           int    `json:"arrival_index"`
	DepartureIndex         int    `json:"departure_index"`
	AllowDischarge         bool   `json:"allow_discharge"`
	FairnessWeight         int64  `json:"fairness_weight"`
}

type Interval struct {
	Index int       `json:"index"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type BaselineInterval struct {
	Index                  int   `json:"index"`
	BaseLoadW              int64 `json:"base_load_w"`
	RenewableW             int64 `json:"renewable_w"`
	NetSiteW               int64 `json:"net_site_w"`
	ImportPriceMicroPerKWh int64 `json:"import_price_micro_per_kwh"`
	ExportPriceMicroPerKWh int64 `json:"export_price_micro_per_kwh"`
	CarbonGramsPerKWh      int64 `json:"carbon_g_per_kwh"`
}

type Baseline struct {
	ScenarioID string             `json:"scenario_id"`
	CreatedAt  time.Time          `json:"created_at"`
	Intervals  []BaselineInterval `json:"intervals"`
}

type Action struct {
	IntervalIndex  int    `json:"interval_index"`
	VehicleID      string `json:"vehicle_id"`
	PowerW         int64  `json:"power_w"`
	GridEnergyWh   int64  `json:"grid_energy_wh"`
	BatteryDeltaWh int64  `json:"battery_delta_wh"`
	SOCBeforeWh    int64  `json:"soc_before_wh"`
	SOCAfterWh     int64  `json:"soc_after_wh"`
	Reason         string `json:"reason"`
}

type SiteInterval struct {
	Index       int   `json:"index"`
	BaseNetW    int64 `json:"base_net_w"`
	FleetPowerW int64 `json:"fleet_power_w"`
	GridPowerW  int64 `json:"grid_power_w"`
	ImportWh    int64 `json:"import_wh"`
	ExportWh    int64 `json:"export_wh"`
}

type VehicleSummary struct {
	VehicleID           string `json:"vehicle_id"`
	InitialSOCWh        int64  `json:"initial_soc_wh"`
	FinalSOCWh          int64  `json:"final_soc_wh"`
	TargetSOCWh         int64  `json:"target_soc_wh"`
	ChargedBatteryWh    int64  `json:"charged_battery_wh"`
	DischargedBatteryWh int64  `json:"discharged_battery_wh"`
	GridImportWh        int64  `json:"grid_import_wh"`
	GridExportWh        int64  `json:"grid_export_wh"`
	FairnessScorePPM    int64  `json:"fairness_score_ppm"`
}

type Plan struct {
	ScenarioID string           `json:"scenario_id"`
	CreatedAt  time.Time        `json:"created_at"`
	Feasible   bool             `json:"feasible"`
	Actions    []Action         `json:"actions"`
	Site       []SiteInterval   `json:"site_intervals"`
	Vehicles   []VehicleSummary `json:"vehicles"`
	Warnings   []string         `json:"warnings"`
}
