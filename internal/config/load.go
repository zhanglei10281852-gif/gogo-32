package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gridflex/internal/model"
)

const MaxInputBytes int64 = 16 << 20

func LoadFile(path string) (model.Scenario, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.Scenario{}, fmt.Errorf("open input: %w", err)
	}
	defer file.Close()
	return Decode(io.LimitReader(file, MaxInputBytes+1))
}

func Decode(reader io.Reader) (model.Scenario, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return model.Scenario{}, fmt.Errorf("read input: %w", err)
	}
	if int64(len(data)) > MaxInputBytes {
		return model.Scenario{}, fmt.Errorf("input exceeds %d bytes", MaxInputBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var scenario model.Scenario
	if err := decoder.Decode(&scenario); err != nil {
		return model.Scenario{}, fmt.Errorf("decode input: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return model.Scenario{}, errors.New("input contains multiple JSON values")
		}
		return model.Scenario{}, fmt.Errorf("decode trailing input: %w", err)
	}
	Normalize(&scenario)
	if err := Validate(scenario); err != nil {
		return model.Scenario{}, err
	}
	return scenario, nil
}

func Normalize(scenario *model.Scenario) {
	scenario.ScenarioID = strings.TrimSpace(scenario.ScenarioID)
	if !scenario.Start.IsZero() {
		scenario.Start = scenario.Start.UTC()
	}
	if scenario.Events == nil {
		scenario.Events = make([]model.SiteCapacityEvent, 0)
	}
	for index := range scenario.Events {
		scenario.Events[index].ID = strings.TrimSpace(scenario.Events[index].ID)
	}
	for index := range scenario.Vehicles {
		scenario.Vehicles[index].ID = strings.TrimSpace(scenario.Vehicles[index].ID)
	}
	sort.SliceStable(scenario.Events, func(i, j int) bool {
		if scenario.Events[i].StartIndex != scenario.Events[j].StartIndex {
			return scenario.Events[i].StartIndex < scenario.Events[j].StartIndex
		}
		if scenario.Events[i].EndIndex != scenario.Events[j].EndIndex {
			return scenario.Events[i].EndIndex < scenario.Events[j].EndIndex
		}
		return scenario.Events[i].ID < scenario.Events[j].ID
	})
	sort.SliceStable(scenario.Vehicles, func(i, j int) bool {
		return scenario.Vehicles[i].ID < scenario.Vehicles[j].ID
	})
	sort.SliceStable(scenario.Tariffs, func(i, j int) bool {
		if scenario.Tariffs[i].StartIndex != scenario.Tariffs[j].StartIndex {
			return scenario.Tariffs[i].StartIndex < scenario.Tariffs[j].StartIndex
		}
		return scenario.Tariffs[i].EndIndex < scenario.Tariffs[j].EndIndex
	})
	sort.SliceStable(scenario.SiteForecast, func(i, j int) bool {
		return scenario.SiteForecast[i].Index < scenario.SiteForecast[j].Index
	})
}

func Validate(scenario model.Scenario) error {
	collector := &validationErrors{}
	validateIdentity(scenario, collector)
	validateHorizon(scenario, collector)
	validateSite(scenario.Site, collector)
	validateEvents(scenario, collector)
	validateOptions(scenario.Options, collector)
	validateTariffs(scenario, collector)
	validateForecast(scenario, collector)
	validateVehicles(scenario, collector)
	if collector.Len() != 0 {
		return collector
	}
	return nil
}

type validationErrors struct {
	items []string
}

func (v *validationErrors) Add(format string, args ...any) {
	v.items = append(v.items, fmt.Sprintf(format, args...))
}

func (v *validationErrors) Len() int {
	return len(v.items)
}

func (v *validationErrors) Error() string {
	return "validation failed: " + strings.Join(v.items, "; ")
}

func validateIdentity(scenario model.Scenario, errs *validationErrors) {
	if scenario.ScenarioID == "" {
		errs.Add("scenario_id is required")
	}
	if len(scenario.ScenarioID) > 128 {
		errs.Add("scenario_id exceeds 128 characters")
	}
	for _, character := range scenario.ScenarioID {
		if character < 32 || character == 127 {
			errs.Add("scenario_id contains a control character")
			break
		}
	}
	if scenario.Start.IsZero() {
		errs.Add("start is required")
	}
	if scenario.Start.Location() == nil {
		errs.Add("start must include a timezone")
	}
}

func validateHorizon(scenario model.Scenario, errs *validationErrors) {
	if scenario.IntervalMinutes <= 0 {
		errs.Add("interval_minutes must be positive")
	}
	if scenario.IntervalMinutes > 24*60 {
		errs.Add("interval_minutes must not exceed 1440")
	}
	if scenario.IntervalCount <= 0 {
		errs.Add("interval_count must be positive")
	}
	if scenario.IntervalCount > 35040 {
		errs.Add("interval_count must not exceed 35040")
	}
	if scenario.IntervalMinutes > 0 && 60%scenario.IntervalMinutes != 0 && scenario.IntervalMinutes%60 != 0 {
		errs.Add("interval_minutes must evenly divide an hour or be whole hours")
	}
}

func validateSite(site model.Site, errs *validationErrors) {
	if site.ImportLimitW <= 0 {
		errs.Add("site.import_limit_w must be positive")
	}
	if site.ExportLimitW < 0 {
		errs.Add("site.export_limit_w must be non-negative")
	}
	if site.ImportLimitW > 10_000_000_000 {
		errs.Add("site.import_limit_w exceeds supported range")
	}
	if site.ExportLimitW > 10_000_000_000 {
		errs.Add("site.export_limit_w exceeds supported range")
	}
}

func validateOptions(options model.Options, errs *validationErrors) {
	if options.FairnessToleranceWh < 0 {
		errs.Add("options.fairness_tolerance_wh must be non-negative")
	}
	if options.ReserveMarginWh < 0 {
		errs.Add("options.reserve_margin_wh must be non-negative")
	}
}
