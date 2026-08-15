package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gridflex/internal/config"
	"gridflex/internal/model"
)

const (
	InputFile      = "input.json"
	BaselineFile   = "baseline.json"
	PlanFile       = "plan.json"
	SettlementFile = "settlement.json"
	AuditFile      = "audit.json"
)

type Store struct {
	dir string
}

func New(dir string) (*Store, error) {
	clean := filepath.Clean(strings.TrimSpace(dir))
	if clean == "." && strings.TrimSpace(dir) == "" {
		return nil, errors.New("store directory is required")
	}
	if err := os.MkdirAll(clean, 0o755); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	return &Store{dir: clean}, nil
}

func (s *Store) Directory() string {
	return s.dir
}

func (s *Store) SaveInput(value model.Scenario) error {
	config.Normalize(&value)
	if err := config.Validate(value); err != nil {
		return fmt.Errorf("validate %s: %w", InputFile, err)
	}
	return s.writeJSON(InputFile, value)
}

func (s *Store) SaveBaseline(value model.Baseline) error {
	return s.writeJSON(BaselineFile, value)
}

func (s *Store) SavePlan(value model.Plan) error {
	return s.writeJSON(PlanFile, value)
}

func (s *Store) SaveSettlement(value model.Settlement) error {
	return s.writeJSON(SettlementFile, value)
}

func (s *Store) SaveAudit(value model.AuditSnapshot) error {
	return s.writeJSON(AuditFile, value)
}

func (s *Store) LoadInput() (model.Scenario, error) {
	var value model.Scenario
	if err := s.readJSON(InputFile, &value); err != nil {
		return model.Scenario{}, err
	}
	config.Normalize(&value)
	if err := config.Validate(value); err != nil {
		return model.Scenario{}, fmt.Errorf("validate %s: %w", InputFile, err)
	}
	return value, nil
}

func (s *Store) LoadBaseline() (model.Baseline, error) {
	var value model.Baseline
	err := s.readJSON(BaselineFile, &value)
	return value, err
}

func (s *Store) LoadPlan() (model.Plan, error) {
	var value model.Plan
	err := s.readJSON(PlanFile, &value)
	return value, err
}

func (s *Store) LoadSettlement() (model.Settlement, error) {
	var value model.Settlement
	err := s.readJSON(SettlementFile, &value)
	return value, err
}

func (s *Store) LoadAudit() (model.AuditSnapshot, error) {
	var value model.AuditSnapshot
	err := s.readJSON(AuditFile, &value)
	return value, err
}

func (s *Store) writeJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	data = append(data, '\n')
	finalPath := filepath.Join(s.dir, name)
	temporary, err := os.CreateTemp(s.dir, "."+name+"-*")
	if err != nil {
		return fmt.Errorf("create temporary %s: %w", name, err)
	}
	temporaryPath := temporary.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("chmod temporary %s: %w", name, err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary %s: %w", name, err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary %s: %w", name, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary %s: %w", name, err)
	}
	_ = os.Remove(finalPath)
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		return fmt.Errorf("replace %s: %w", name, err)
	}
	remove = false
	return nil
}

func (s *Store) readJSON(name string, destination any) error {
	file, err := os.Open(filepath.Join(s.dir, name))
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s contains trailing JSON", name)
	}
	return nil
}
