package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type ArtifactInfo struct {
	Name       string
	Path       string
	Bytes      int64
	ModifiedAt time.Time
	SHA256     string
}

func (s *Store) Catalog() ([]ArtifactInfo, error) {
	names := []string{InputFile, BaselineFile, PlanFile, SettlementFile, AuditFile}
	result := make([]ArtifactInfo, 0, len(names))
	for _, name := range names {
		path := filepath.Join(s.dir, name)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat artifact %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("artifact %s is not a regular file", name)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read artifact %s: %w", name, err)
		}
		result = append(result, ArtifactInfo{
			Name:       name,
			Path:       path,
			Bytes:      info.Size(),
			ModifiedAt: info.ModTime().UTC(),
			SHA256:     HashBytes(data),
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *Store) Complete() (bool, error) {
	catalog, err := s.Catalog()
	if err != nil {
		return false, err
	}
	if len(catalog) != 5 {
		return false, nil
	}
	expected := map[string]bool{
		InputFile:      true,
		BaselineFile:   true,
		PlanFile:       true,
		SettlementFile: true,
		AuditFile:      true,
	}
	for _, artifact := range catalog {
		if !expected[artifact.Name] || artifact.Bytes == 0 {
			return false, nil
		}
		delete(expected, artifact.Name)
	}
	return len(expected) == 0, nil
}

func (s *Store) RemoveArtifacts() error {
	names := []string{InputFile, BaselineFile, PlanFile, SettlementFile, AuditFile}
	for _, name := range names {
		err := os.Remove(filepath.Join(s.dir, name))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove artifact %s: %w", name, err)
		}
	}
	return nil
}

func (s *Store) ArtifactPath(name string) (string, error) {
	switch name {
	case InputFile, BaselineFile, PlanFile, SettlementFile, AuditFile:
		return filepath.Join(s.dir, name), nil
	default:
		return "", fmt.Errorf("unknown artifact %q", name)
	}
}
