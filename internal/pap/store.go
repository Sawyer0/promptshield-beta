package pap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/uuid"
)

// Store defines persistence for bundles.
type Store interface {
	Save(ctx context.Context, b Bundle) (string, error)
	Load(ctx context.Context, tenantID, rulepackID uuid.UUID, version int) (Bundle, error)
	List(ctx context.Context, tenantID, rulepackID uuid.UUID) ([]BundleInfo, error)
}

type BundleInfo struct {
	Version   int    `json:"version"`
	KeyID     string `json:"key_id,omitempty"`
	Checksum  string `json:"checksum_sha256"`
	CreatedAt string `json:"created_at"`
	Path      string `json:"path"`
}

// FSStore stores bundles under baseDir/<tenant>/<rulepack>/v<version>.json
// It is intended for dev/test and simple on-prem deployments.
type FSStore struct{ baseDir string }

func NewFSStore(baseDir string) (*FSStore, error) {
	if baseDir == "" {
		return nil, errors.New("baseDir is required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}
	return &FSStore{baseDir: baseDir}, nil
}

func (s *FSStore) Save(ctx context.Context, b Bundle) (string, error) {
	path := s.bundlePath(b.TenantID, b.RulepackID, b.Version)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(b); err != nil {
		return "", err
	}
	return path, nil
}

func (s *FSStore) Load(ctx context.Context, tenantID, rulepackID uuid.UUID, version int) (Bundle, error) {
	var b Bundle
	path := s.bundlePath(tenantID, rulepackID, version)
	data, err := os.ReadFile(path)
	if err != nil {
		return b, err
	}
	if err := json.Unmarshal(data, &b); err != nil {
		return b, err
	}
	return b, nil
}

func (s *FSStore) List(ctx context.Context, tenantID, rulepackID uuid.UUID) ([]BundleInfo, error) {
	dir := filepath.Join(s.baseDir, tenantID.String(), rulepackID.String())
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BundleInfo{}, nil
		}
		return nil, err
	}
	var infos []BundleInfo
	for _, e := range entries {
		if e.IsDir() { continue }
		if filepath.Ext(e.Name()) != ".json" { continue }
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil { continue }
		var b Bundle
		if err := json.Unmarshal(data, &b); err != nil { continue }
		infos = append(infos, BundleInfo{Version: b.Version, KeyID: b.KeyID, Checksum: b.Checksum, CreatedAt: b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), Path: path})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Version < infos[j].Version })
	return infos, nil
}

func (s *FSStore) bundlePath(tenantID, rulepackID uuid.UUID, version int) string {
	return filepath.Join(s.baseDir, tenantID.String(), rulepackID.String(), fmt.Sprintf("v%d.json", version))
}

