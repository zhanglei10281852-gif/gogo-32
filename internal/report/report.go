package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"gridflex/internal/model"
)

type Format string

const (
	Text Format = "text"
	JSON Format = "json"
)

func ParseFormat(value string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "text", "txt":
		return Text, nil
	case "json":
		return JSON, nil
	default:
		return "", fmt.Errorf("unsupported report format %q", value)
	}
}

func Document(plan model.Plan, settlement model.Settlement, audit model.AuditSnapshot) (model.ReportDocument, error) {
	if plan.ScenarioID == "" {
		return model.ReportDocument{}, errors.New("plan scenario ID is required")
	}
	if settlement.ScenarioID != plan.ScenarioID {
		return model.ReportDocument{}, errors.New("settlement scenario ID does not match plan")
	}
	if audit.ScenarioID != plan.ScenarioID {
		return model.ReportDocument{}, errors.New("audit scenario ID does not match plan")
	}
	vehicles := append([]model.VehicleSummary(nil), plan.Vehicles...)
	sort.SliceStable(vehicles, func(i, j int) bool {
		return vehicles[i].VehicleID < vehicles[j].VehicleID
	})
	warnings := append([]string(nil), plan.Warnings...)
	sort.Strings(warnings)
	return model.ReportDocument{
		ScenarioID: plan.ScenarioID,
		Feasible:   plan.Feasible,
		Settlement: settlement,
		Vehicles:   vehicles,
		Warnings:   warnings,
		Audit:      audit,
	}, nil
}

func Write(writer io.Writer, format Format, document model.ReportDocument) error {
	switch format {
	case JSON:
		return writeJSON(writer, document)
	case Text:
		return writeText(writer, document)
	default:
		return fmt.Errorf("unsupported report format %q", format)
	}
}

func writeJSON(writer io.Writer, document model.ReportDocument) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("encode JSON report: %w", err)
	}
	return nil
}

func writeText(writer io.Writer, document model.ReportDocument) error {
	lines := make([]string, 0, 32+len(document.Vehicles)+len(document.Warnings))
	lines = append(lines, "GRIDFLEX REPORT")
	lines = append(lines, "scenario: "+document.ScenarioID)
	lines = append(lines, fmt.Sprintf("feasible: %t", document.Feasible))
	lines = append(lines, "")
	lines = append(lines, "ENERGY AND VALUE")
	lines = append(lines, fmt.Sprintf("grid import: %d Wh", document.Settlement.TotalImportWh))
	lines = append(lines, fmt.Sprintf("grid export: %d Wh", document.Settlement.TotalExportWh))
	lines = append(lines, fmt.Sprintf("import cost: %d micro", document.Settlement.TotalImportCostMicro))
	lines = append(lines, fmt.Sprintf("export revenue: %d micro", document.Settlement.TotalExportRevenueMicro))
	lines = append(lines, fmt.Sprintf("net cost: %d micro", document.Settlement.NetCostMicro))
	lines = append(lines, fmt.Sprintf("baseline cost: %d micro", document.Settlement.BaselineCostMicro))
	lines = append(lines, fmt.Sprintf("savings: %d micro", document.Settlement.SavingsMicro))
	lines = append(lines, fmt.Sprintf("carbon: %d g", document.Settlement.CarbonGrams))
	lines = append(lines, fmt.Sprintf("baseline carbon: %d g", document.Settlement.BaselineCarbonGrams))
	lines = append(lines, fmt.Sprintf("carbon delta: %d g", document.Settlement.CarbonDeltaGrams))
	lines = append(lines, "")
	lines = append(lines, "VEHICLES")
	for _, vehicle := range document.Vehicles {
		lines = append(lines, fmt.Sprintf("%s initial=%dWh final=%dWh target=%dWh charge=%dWh discharge=%dWh fairness=%dppm", vehicle.VehicleID, vehicle.InitialSOCWh, vehicle.FinalSOCWh, vehicle.TargetSOCWh, vehicle.ChargedBatteryWh, vehicle.DischargedBatteryWh, vehicle.FairnessScorePPM))
	}
	lines = append(lines, "")
	lines = append(lines, "AUDIT")
	lines = append(lines, "algorithm: "+document.Audit.Algorithm)
	lines = append(lines, "algorithm version: "+document.Audit.AlgorithmVersion)
	lines = append(lines, "input sha256: "+document.Audit.InputSHA256)
	lines = append(lines, "result sha256: "+document.Audit.ResultSHA256)
	lines = append(lines, "root: "+document.Audit.RootSHA256)
	lines = append(lines, fmt.Sprintf("config: start=%s interval_minutes=%d interval_count=%d events=%d", document.Audit.Config.Start.Format(time.RFC3339), document.Audit.Config.IntervalMinutes, document.Audit.Config.IntervalCount, len(document.Audit.Config.Events)))
	lines = append(lines, fmt.Sprintf("config site: import_limit_w=%d export_limit_w=%d", document.Audit.Config.Site.ImportLimitW, document.Audit.Config.Site.ExportLimitW))
	for _, event := range document.Audit.Config.Events {
		lines = append(lines, fmt.Sprintf("event %s [%d,%d) import_limit_w=%d export_limit_w=%d", event.ID, event.StartIndex, event.EndIndex, event.ImportLimitW, event.ExportLimitW))
	}
	for _, entry := range document.Audit.Entries {
		lines = append(lines, fmt.Sprintf("%s bytes=%d sha256=%s", entry.Name, entry.Bytes, entry.SHA256))
	}
	if len(document.Warnings) != 0 {
		lines = append(lines, "")
		lines = append(lines, "WARNINGS")
		for _, warning := range document.Warnings {
			lines = append(lines, "- "+warning)
		}
	}
	_, err := io.WriteString(writer, strings.Join(lines, "\n")+"\n")
	if err != nil {
		return fmt.Errorf("write text report: %w", err)
	}
	return nil
}
