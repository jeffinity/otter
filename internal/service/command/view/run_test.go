package view

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffinity/otter/internal/otterfs"
	"github.com/jeffinity/otter/internal/service/command/servicefile"
)

func TestRunShowsClassicService(t *testing.T) {
	servicePath := writeFile(t, "api.service", "\n[Unit]\nDescription=API\n\n")
	out, err := runView(t, "api.service", fakeFinder{file: servicefile.File{Name: "api", Path: servicePath, Source: servicefile.SourceClassic}})
	if err != nil {
		t.Fatalf("run view: %v", err)
	}
	if got, want := out, "[Unit]\nDescription=API\n"; got != want {
		t.Fatalf("classic output = %q, want %q", got, want)
	}
}

func TestRunTrimsPackageHeader(t *testing.T) {
	servicePath := writeFile(t, "api.service", "# generated\n# do not edit\n\n[Unit]\nDescription=API\n")
	out, err := runView(t, "api", fakeFinder{file: servicefile.File{Name: "api", Path: servicePath, Source: servicefile.SourcePackage}})
	if err != nil {
		t.Fatalf("run view: %v", err)
	}
	if got, want := out, "[Unit]\nDescription=API\n"; got != want {
		t.Fatalf("package output = %q, want %q", got, want)
	}
}

func TestRunShowsDockerCompose(t *testing.T) {
	dir := t.TempDir()
	composeDir := filepath.Join(dir, "compose")
	if err := os.MkdirAll(composeDir, 0o755); err != nil {
		t.Fatalf("mkdir compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(composeDir, "docker-compose.yml"), []byte("\nservices:\n  api:\n    image: api\n\n"), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	servicePath := filepath.Join(dir, "api.service")
	data := "[Unit]\nDescription=API\n\n[X-Otter]\nDockerComposeBaseDir=" + composeDir + "\n"
	if err := os.WriteFile(servicePath, []byte(data), 0o644); err != nil {
		t.Fatalf("write service: %v", err)
	}

	out, err := runView(t, "api", fakeFinder{file: servicefile.File{Name: "api", Path: servicePath, Source: servicefile.SourceClassic}})
	if err != nil {
		t.Fatalf("run view: %v", err)
	}
	for _, needle := range []string{
		"#==========================\n",
		"# Docker Compose: " + filepath.Join(composeDir, "docker-compose.yml") + "\n",
		"# services:\n",
		"#   api:\n",
		"#     image: api\n",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("compose output missing %q: %s", needle, out)
		}
	}
}

func TestRunReturnsFindError(t *testing.T) {
	_, err := runView(t, "missing", fakeFinder{err: os.ErrNotExist})
	if err == nil {
		t.Fatalf("expected find error")
	}
}

func TestFSFinderFindsClassicBeforePackage(t *testing.T) {
	dir := t.TempDir()
	classic := filepath.Join(dir, "classic")
	pkg := filepath.Join(dir, "packages", "pkg", "api")
	if err := os.MkdirAll(classic, 0o755); err != nil {
		t.Fatalf("mkdir classic: %v", err)
	}
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatalf("mkdir package: %v", err)
	}
	classicPath := filepath.Join(classic, "api.service")
	if err := os.WriteFile(classicPath, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatalf("write classic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "api.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatalf("write package: %v", err)
	}

	finder := servicefile.FSFinder{FS: otterfs.New(otterfs.Config{
		ClassicServicePath: classic,
		PackageServicePath: filepath.Join(dir, "packages"),
	})}
	file, err := finder.Find(context.Background(), "api.service")
	if err != nil {
		t.Fatalf("find service: %v", err)
	}
	if file.Path != classicPath || file.Source != servicefile.SourceClassic {
		t.Fatalf("file = %+v, want classic %s", file, classicPath)
	}
}

func runView(t *testing.T, serviceName string, finder servicefile.Finder) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(context.Background(), serviceName, Options{NoColor: true}, Dependencies{
		Finder: finder,
		Out:    &out,
	})
	return out.String(), err
}

func writeFile(t *testing.T, name string, data string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return p
}

type fakeFinder struct {
	file servicefile.File
	err  error
}

func (f fakeFinder) Find(ctx context.Context, serviceName string) (servicefile.File, error) {
	return f.file, f.err
}
