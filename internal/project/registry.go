package project

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Resetnak/herdr-logbook/internal/storage"
	"github.com/pelletier/go-toml/v2"
)

type Registry struct {
	Version  int             `toml:"version"`
	Projects []ProjectRecord `toml:"projects"`
}

type ProjectRecord struct {
	ID                string    `toml:"id"`
	Name              string    `toml:"name"`
	Storage           string    `toml:"storage"`
	StorePath         string    `toml:"store_path"`
	FirstSeen         time.Time `toml:"first_seen"`
	LastSeen          time.Time `toml:"last_seen"`
	RemoteFingerprint string    `toml:"remote_fingerprint,omitempty"`
	DefaultRoot       string    `toml:"default_root"`
	Roots             []string  `toml:"roots"`
}

func LoadRegistry(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Registry{Version: 1}, nil
	}
	if err != nil {
		return Registry{}, fmt.Errorf("read registry %q: %w", path, err)
	}
	var registry Registry
	if err := toml.Unmarshal(data, &registry); err != nil {
		return Registry{}, fmt.Errorf("decode registry %q: %w", path, err)
	}
	if registry.Version == 0 {
		registry.Version = 1
	}
	if registry.Version != 1 {
		return Registry{}, fmt.Errorf("unsupported registry version %d", registry.Version)
	}
	return registry, nil
}

func (r *Registry) Upsert(project Project, storageMode, storePath string, now time.Time) {
	for i := range r.Projects {
		if r.Projects[i].ID == project.ID {
			record := &r.Projects[i]
			record.Name = project.Name
			record.Storage = storageMode
			record.StorePath = storePath
			record.LastSeen = now
			record.RemoteFingerprint = project.Fingerprint
			record.DefaultRoot = project.Root
			record.Roots = mergeRoots(record.Roots, append(project.Roots, project.Root)...)
			return
		}
	}
	r.Projects = append(r.Projects, ProjectRecord{
		ID: project.ID, Name: project.Name, Storage: storageMode, StorePath: storePath,
		FirstSeen: now, LastSeen: now, RemoteFingerprint: project.Fingerprint,
		DefaultRoot: project.Root, Roots: mergeRoots(nil, append(project.Roots, project.Root)...),
	})
	sort.Slice(r.Projects, func(i, j int) bool { return r.Projects[i].ID < r.Projects[j].ID })
}

func UpdateRegistry(path, lockPath string, timeout time.Duration, project Project, storageMode, storePath string, now time.Time) error {
	return storage.WithLock(lockPath, timeout, func() error {
		registry, err := LoadRegistry(path)
		if err != nil {
			return err
		}
		registry.Upsert(project, storageMode, storePath, now)
		data, err := toml.Marshal(registry)
		if err != nil {
			return fmt.Errorf("encode registry: %w", err)
		}
		return storage.AtomicWrite(path, data, 0o600)
	})
}

func mergeRoots(existing []string, roots ...string) []string {
	unique := make(map[string]struct{}, len(existing)+len(roots))
	for _, root := range append(existing, roots...) {
		if root != "" {
			unique[root] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for root := range unique {
		result = append(result, root)
	}
	sort.Strings(result)
	return result
}
