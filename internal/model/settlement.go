package model

import "time"

type SettlementInterval struct {
	Index               int   `json:"index"`
	ImportWh            int64 `json:"import_wh"`
	ExportWh            int64 `json:"export_wh"`
	ImportCostMicro     int64 `json:"import_cost_micro"`
	ExportRevenueMicro  int64 `json:"export_revenue_micro"`
	NetCostMicro        int64 `json:"net_cost_micro"`
	CarbonGrams         int64 `json:"carbon_grams"`
	BaselineCostMicro   int64 `json:"baseline_cost_micro"`
	BaselineCarbonGrams int64 `json:"baseline_carbon_grams"`
}

type VehicleSettlement struct {
	VehicleID          string `json:"vehicle_id"`
	ImportWh           int64  `json:"import_wh"`
	ExportWh           int64  `json:"export_wh"`
	EnergyCostMicro    int64  `json:"energy_cost_micro"`
	ExportRevenueMicro int64  `json:"export_revenue_micro"`
	NetCostMicro       int64  `json:"net_cost_micro"`
	CarbonGrams        int64  `json:"carbon_grams"`
}

type Settlement struct {
	ScenarioID              string               `json:"scenario_id"`
	CreatedAt               time.Time            `json:"created_at"`
	Intervals               []SettlementInterval `json:"intervals"`
	Vehicles                []VehicleSettlement  `json:"vehicles"`
	TotalImportWh           int64                `json:"total_import_wh"`
	TotalExportWh           int64                `json:"total_export_wh"`
	TotalImportCostMicro    int64                `json:"total_import_cost_micro"`
	TotalExportRevenueMicro int64                `json:"total_export_revenue_micro"`
	NetCostMicro            int64                `json:"net_cost_micro"`
	CarbonGrams             int64                `json:"carbon_grams"`
	BaselineCostMicro       int64                `json:"baseline_cost_micro"`
	BaselineCarbonGrams     int64                `json:"baseline_carbon_grams"`
	SavingsMicro            int64                `json:"savings_micro"`
	CarbonDeltaGrams        int64                `json:"carbon_delta_grams"`
}

type AuditEntry struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type AuditSnapshot struct {
	ScenarioID       string       `json:"scenario_id"`
	Algorithm        string       `json:"algorithm"`
	InputSHA256      string       `json:"input_sha256"`
	Config           Scenario     `json:"config"`
	AlgorithmVersion string       `json:"algorithm_version"`
	ResultSHA256     string       `json:"result_sha256"`
	Entries          []AuditEntry `json:"entries"`
	RootSHA256       string       `json:"root_sha256"`
}

type ReportDocument struct {
	ScenarioID string           `json:"scenario_id"`
	Feasible   bool             `json:"feasible"`
	Settlement Settlement       `json:"settlement"`
	Vehicles   []VehicleSummary `json:"vehicles"`
	Warnings   []string         `json:"warnings"`
	Audit      AuditSnapshot    `json:"audit"`
}
