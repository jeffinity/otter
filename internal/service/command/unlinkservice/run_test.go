package unlinkservice

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffinity/otter/internal/otterfs"
)

func TestRunRemovesLinkedService(t *testing.T) {
	root := t.TempDir()
	unit := filepath.Join(root, "systemd", "api.service")
	if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unit, []byte("[Service]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	classic := filepath.Join(root, "classic")
	if err := os.MkdirAll(classic, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(classic, "api.service")
	if err := os.Symlink(unit, link); err != nil {
		t.Fatal(err)
	}
	fs := otterfs.New(otterfs.Config{ClassicServicePath: classic})
	var out bytes.Buffer

	err := Run(context.Background(), "api.service", Dependencies{
		FS:  fs,
		Out: &out,
	})
	if err != nil {
		t.Fatalf("run unlink-service: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("link should be removed, got %v", err)
	}
	if _, err := os.Stat(unit); err != nil {
		t.Fatalf("unit should stay untouched: %v", err)
	}
	if !strings.Contains(out.String(), "Remove fake service link "+link+" success.\n") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunRejectsMissingService(t *testing.T) {
	root := t.TempDir()
	err := Run(context.Background(), "api", Dependencies{
		FS: otterfs.New(otterfs.Config{ClassicServicePath: filepath.Join(root, "classic")}),
	})
	if err == nil || err.Error() != "service api is not linked by otter" {
		t.Fatalf("expected missing error, got %v", err)
	}
}

func TestRunRejectsClassicFile(t *testing.T) {
	root := t.TempDir()
	classic := filepath.Join(root, "classic")
	if err := os.MkdirAll(classic, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(classic, "api.service")
	if err := os.WriteFile(file, []byte("[Service]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(context.Background(), "api", Dependencies{
		FS: otterfs.New(otterfs.Config{ClassicServicePath: classic}),
	})
	if err == nil || err.Error() != "service api is managed by otter but not linked" {
		t.Fatalf("expected regular file error, got %v", err)
	}
}
