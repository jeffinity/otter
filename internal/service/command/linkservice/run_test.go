package linkservice

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffinity/otter/internal/otterfs"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

func TestRunLinksExistingUnit(t *testing.T) {
	root := t.TempDir()
	unit := filepath.Join(root, "systemd", "api.service")
	if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unit, []byte("[Service]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := otterfs.New(otterfs.Config{ClassicServicePath: filepath.Join(root, "classic")})
	var out bytes.Buffer

	err := Run(context.Background(), "api.service", Dependencies{
		Store: fakeStore{services: []statuscmd.Service{{Name: "api", FragmentPath: unit}}},
		FS:    fs,
		Out:   &out,
	})
	if err != nil {
		t.Fatalf("run link-service: %v", err)
	}
	target, err := os.Readlink(filepath.Join(root, "classic", "api.service"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != unit {
		t.Fatalf("link target = %q, want %q", target, unit)
	}
	if !strings.Contains(out.String(), "Create fake service (linked to "+unit+") success.\n") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunRejectsCore(t *testing.T) {
	err := Run(context.Background(), "otter-core", Dependencies{})
	if err == nil || err.Error() != "otter-core must be managed outside of otter service" {
		t.Fatalf("expected core error, got %v", err)
	}
}

type fakeStore struct {
	services []statuscmd.Service
}

func (f fakeStore) List(ctx context.Context) ([]statuscmd.Service, error) {
	return f.services, nil
}
