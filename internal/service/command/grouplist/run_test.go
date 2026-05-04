package grouplist

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jeffinity/otter/internal/otterfs"
)

func TestRunListsGroups(t *testing.T) {
	var out bytes.Buffer
	err := Run(context.Background(), Options{}, Dependencies{
		Store: fakeStore{groups: map[string][]string{
			"web":    {"api"},
			"worker": {"job"},
		}},
		Out: &out,
	})
	if err != nil {
		t.Fatalf("run group-list: %v", err)
	}
	if got, want := out.String(), "web\nworker\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunListsGroupsOneLine(t *testing.T) {
	var out bytes.Buffer
	err := Run(context.Background(), Options{OneLine: true}, Dependencies{
		Store: fakeStore{groups: map[string][]string{
			"worker": {"job"},
			"web":    {"api"},
		}},
		Out: &out,
	})
	if err != nil {
		t.Fatalf("run group-list --one: %v", err)
	}
	if got, want := out.String(), "web worker\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunListsServices(t *testing.T) {
	var out bytes.Buffer
	err := Run(context.Background(), Options{OneLine: true, IncludeServices: true}, Dependencies{
		Store: fakeStore{groups: map[string][]string{
			"web": {"job", "api"},
		}},
		Out: &out,
	})
	if err != nil {
		t.Fatalf("run group-list --services: %v", err)
	}
	if got, want := out.String(), "web: api, job\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunReturnsStoreError(t *testing.T) {
	err := Run(context.Background(), Options{}, Dependencies{
		Store: fakeStore{err: errors.New("boom")},
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestFSStoreListsGroups(t *testing.T) {
	root := t.TempDir()
	classic := filepath.Join(root, "classic")
	pkg := filepath.Join(root, "pkg")
	writeService(t, filepath.Join(classic, "api.service"), "[X-Otter]\nGroup=web, edge\n")
	writeService(t, filepath.Join(pkg, "bundle", "services", "job.service"), "[X-Otter]\nGroup=worker\nGroup=web\n")

	groups, err := FSStore{FS: otterfs.New(otterfs.Config{
		ClassicServicePath: classic,
		PackageServicePath: pkg,
	})}.List(context.Background())
	if err != nil {
		t.Fatalf("list fs groups: %v", err)
	}
	if got, want := stringsOf(groups["web"]), "api,job"; got != want {
		t.Fatalf("web group = %q, want %q", got, want)
	}
	if got, want := stringsOf(groups["edge"]), "api"; got != want {
		t.Fatalf("edge group = %q, want %q", got, want)
	}
	if got, want := stringsOf(groups["worker"]), "job"; got != want {
		t.Fatalf("worker group = %q, want %q", got, want)
	}
}

type fakeStore struct {
	groups map[string][]string
	err    error
}

func (f fakeStore) List(ctx context.Context) (map[string][]string, error) {
	return f.groups, f.err
}

func writeService(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write service: %v", err)
	}
}

func stringsOf(values []string) string {
	sort.Strings(values)
	return strings.Join(values, ",")
}
