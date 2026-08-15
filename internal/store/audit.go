package store

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"gridflex/internal/model"
)

const (
	AuditAlgorithm   = "SHA-256"
	AlgorithmVersion = "gridflex-dispatch-v1"
)

type namedValue struct {
	name  string
	value any
}

type auditRootPayload struct {
	ScenarioID       string             `json:"scenario_id"`
	Algorithm        string             `json:"algorithm"`
	InputSHA256      string             `json:"input_sha256"`
	Config           model.Scenario     `json:"config"`
	AlgorithmVersion string             `json:"algorithm_version"`
	ResultSHA256     string             `json:"result_sha256"`
	Entries          []model.AuditEntry `json:"entries"`
}

func (s *Store) CreateAudit() (model.AuditSnapshot, error) {
	scenario, err := s.LoadInput()
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	baseline, err := s.LoadBaseline()
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	plan, err := s.LoadPlan()
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	settled, err := s.LoadSettlement()
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	if baseline.ScenarioID != scenario.ScenarioID || plan.ScenarioID != scenario.ScenarioID || settled.ScenarioID != scenario.ScenarioID {
		return model.AuditSnapshot{}, fmt.Errorf("result scenario IDs do not match input %q", scenario.ScenarioID)
	}

	inputJSON, err := json.Marshal(scenario)
	if err != nil {
		return model.AuditSnapshot{}, fmt.Errorf("canonicalize input: %w", err)
	}
	resultSHA256, err := hashNamedValues([]namedValue{
		{name: BaselineFile, value: baseline},
		{name: PlanFile, value: plan},
		{name: SettlementFile, value: settled},
	})
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	entries, err := s.auditEntries([]string{InputFile, BaselineFile, PlanFile, SettlementFile})
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	snapshot := model.AuditSnapshot{
		ScenarioID:       scenario.ScenarioID,
		Algorithm:        AuditAlgorithm,
		InputSHA256:      HashBytes(inputJSON),
		Config:           scenario,
		AlgorithmVersion: AlgorithmVersion,
		ResultSHA256:     resultSHA256,
		Entries:          entries,
	}
	snapshot.RootSHA256, err = auditRootSHA256(snapshot)
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Store) auditEntries(names []string) ([]model.AuditEntry, error) {
	ordered := append([]string(nil), names...)
	sort.Strings(ordered)
	entries := make([]model.AuditEntry, 0, len(ordered))
	for _, name := range ordered {
		data, err := os.ReadFile(filepath.Join(s.dir, name))
		if err != nil {
			return nil, fmt.Errorf("read audit artifact %s: %w", name, err)
		}
		entries = append(entries, model.AuditEntry{Name: name, SHA256: HashBytes(data), Bytes: int64(len(data))})
	}
	return entries, nil
}

func hashNamedValues(values []namedValue) (string, error) {
	hash := sha256.New()
	for _, item := range values {
		data, err := json.Marshal(item.value)
		if err != nil {
			return "", fmt.Errorf("canonicalize %s: %w", item.name, err)
		}
		writeFrame(hash, []byte(item.name))
		writeFrame(hash, data)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeFrame(hash interface{ Write([]byte) (int, error) }, data []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(data)
}

func auditRootSHA256(snapshot model.AuditSnapshot) (string, error) {
	payload := auditRootPayload{
		ScenarioID:       snapshot.ScenarioID,
		Algorithm:        snapshot.Algorithm,
		InputSHA256:      snapshot.InputSHA256,
		Config:           snapshot.Config,
		AlgorithmVersion: snapshot.AlgorithmVersion,
		ResultSHA256:     snapshot.ResultSHA256,
		Entries:          snapshot.Entries,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode audit root payload: %w", err)
	}
	return HashBytes(data), nil
}

func (s *Store) Snapshot() (model.AuditSnapshot, error) {
	snapshot, err := s.CreateAudit()
	if err != nil {
		return model.AuditSnapshot{}, err
	}
	if err := s.SaveAudit(snapshot); err != nil {
		return model.AuditSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Store) VerifyAudit() error {
	expected, err := s.LoadAudit()
	if err != nil {
		return err
	}
	if expected.Algorithm != AuditAlgorithm {
		return fmt.Errorf("unsupported audit algorithm %q", expected.Algorithm)
	}
	if expected.AlgorithmVersion != AlgorithmVersion {
		return fmt.Errorf("unsupported algorithm version %q", expected.AlgorithmVersion)
	}
	actual, err := s.CreateAudit()
	if err != nil {
		return err
	}
	if expected.ScenarioID != actual.ScenarioID {
		return fmt.Errorf("audit scenario ID mismatch: expected %q, got %q", expected.ScenarioID, actual.ScenarioID)
	}
	if expected.InputSHA256 != actual.InputSHA256 {
		return fmt.Errorf("audit input hash mismatch: expected %s, got %s", expected.InputSHA256, actual.InputSHA256)
	}
	if !reflect.DeepEqual(expected.Config, actual.Config) {
		return fmt.Errorf("audit config does not match normalized input")
	}
	if expected.ResultSHA256 != actual.ResultSHA256 {
		return fmt.Errorf("audit result hash mismatch: expected %s, got %s", expected.ResultSHA256, actual.ResultSHA256)
	}
	if len(expected.Entries) != len(actual.Entries) {
		return fmt.Errorf("audit entry count mismatch")
	}
	for index := range expected.Entries {
		left := expected.Entries[index]
		right := actual.Entries[index]
		if left.Name != right.Name {
			return fmt.Errorf("audit entry %d name mismatch", index)
		}
		if left.SHA256 != right.SHA256 {
			return fmt.Errorf("audit digest mismatch for %s", left.Name)
		}
		if left.Bytes != right.Bytes {
			return fmt.Errorf("audit size mismatch for %s", left.Name)
		}
	}
	if expected.RootSHA256 != actual.RootSHA256 {
		return fmt.Errorf("audit root mismatch: expected %s, got %s", expected.RootSHA256, actual.RootSHA256)
	}
	return nil
}

func HashBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func SortedEntries(entries []model.AuditEntry) []model.AuditEntry {
	result := append([]model.AuditEntry(nil), entries...)
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
