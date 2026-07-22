package project

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRegistryUpsertPreservesFirstSeenAndAddsMovedRoot(t *testing.T) {
	first := time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	project := Project{ID: "p_one", Name: "one", Root: "/old/root", Roots: []string{"/old/root"}, Fingerprint: "github.com/org/repo"}
	registry := Registry{Version: 1}
	registry.Upsert(project, "central", "/store/p_one", first)
	project.Root = "/new/root"
	project.Roots = []string{"/new/root"}
	registry.Upsert(project, "central", "/store/p_one", second)

	if len(registry.Projects) != 1 {
		t.Fatalf("Upsert() projects = %#v", registry.Projects)
	}
	record := registry.Projects[0]
	if !record.FirstSeen.Equal(first) || !record.LastSeen.Equal(second) || record.DefaultRoot != "/new/root" {
		t.Fatalf("Upsert() record = %#v", record)
	}
	if len(record.Roots) != 2 || record.Roots[0] != "/new/root" || record.Roots[1] != "/old/root" {
		t.Fatalf("Upsert() roots = %#v", record.Roots)
	}
}

func TestUpdateRegistryWritesCredentialFreeTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry", "projects.toml")
	lock := filepath.Join(dir, "locks", "registry.lock")
	project := Project{ID: "p_one", Name: "one", Root: "/repo", Roots: []string{"/repo"}, Fingerprint: "github.com/org/repo"}
	if err := UpdateRegistry(path, lock, time.Second, project, "central", "/store/p_one", time.Now().UTC()); err != nil {
		t.Fatalf("UpdateRegistry() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "password") || strings.Contains(string(data), "@github") {
		t.Fatalf("UpdateRegistry() leaked credentials: %s", data)
	}
	loaded, err := LoadRegistry(path)
	if err != nil || len(loaded.Projects) != 1 || loaded.Projects[0].ID != "p_one" {
		t.Fatalf("LoadRegistry() = %#v, %v", loaded, err)
	}
}

func TestUpdateRegistryConcurrentProjectsAreNotLost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry", "projects.toml")
	lock := filepath.Join(dir, "locks", "registry.lock")
	var wg sync.WaitGroup
	for _, id := range []string{"p_one", "p_two"} {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			project := Project{ID: id, Name: id, Root: "/" + id, Roots: []string{"/" + id}, Fingerprint: "github.com/org/" + id}
			if err := UpdateRegistry(path, lock, time.Second, project, "central", "/store/"+id, time.Now().UTC()); err != nil {
				t.Errorf("UpdateRegistry(%s) error = %v", id, err)
			}
		}()
	}
	wg.Wait()
	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Projects) != 2 {
		t.Fatalf("concurrent UpdateRegistry() projects = %#v", loaded.Projects)
	}
}

func TestLoadRegistryRejectsCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.toml")
	if err := os.WriteFile(path, []byte("[[projects]\ninvalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistry(path); err == nil || !strings.Contains(err.Error(), "decode registry") {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
}
