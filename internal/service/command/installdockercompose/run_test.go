package installdockercompose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCopiesComposeAndInstalls(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "stack")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "docker-compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installer := &fakeInstaller{}

	baseDir := filepath.Join(root, "opt", "stack")
	err := Run(context.Background(), []string{sourceDir}, Options{BaseDir: baseDir}, Dependencies{
		Installer: installer,
		LookPath:  func(file string) (string, error) { return "/usr/bin/docker", nil },
	})
	if err != nil {
		t.Fatalf("run idc: %v", err)
	}
	if installer.name != "stack" {
		t.Fatalf("installer name = %q, want stack", installer.name)
	}
	if !strings.Contains(string(installer.data), "[X-Otter]\nDockerComposeBaseDir="+baseDir) {
		t.Fatalf("service data = %q", string(installer.data))
	}
}

func TestCopyToRejectsExistingWithoutForce(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "from")
	to := filepath.Join(dir, "to")
	_ = os.WriteFile(from, []byte("from"), 0o644)
	_ = os.WriteFile(to, []byte("to"), 0o644)

	if err := copyTo(from, to, false); err == nil {
		t.Fatalf("expected existing file error")
	}
	if err := copyTo(from, to, true); err != nil {
		t.Fatalf("force copy: %v", err)
	}
}

type fakeInstaller struct {
	name string
	data []byte
}

func (f *fakeInstaller) Install(ctx context.Context, name string, data []byte) error {
	f.name = name
	f.data = append([]byte(nil), data...)
	return nil
}
