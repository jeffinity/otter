package installservice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffinity/otter/internal/otterfs"
)

func TestRunInstallsAndStarts(t *testing.T) {
	file := filepath.Join(t.TempDir(), "api.service")
	write(t, file, "[Unit]\nDescription=API\n[Service]\nExecStart=/bin/api\n")
	installer := &fakeInstaller{}
	runner := &fakeRunner{}
	var out bytes.Buffer

	err := Run(context.Background(), file, Options{}, Dependencies{
		Installer: installer,
		Runner:    runner,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("run install-service: %v", err)
	}
	if installer.name != "api" {
		t.Fatalf("installer name = %q, want api", installer.name)
	}
	if got, want := strings.Join(runner.calls[0], " "), "daemon-reload"; got != want {
		t.Fatalf("call 0 = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.calls[1], " "), "enable api"; got != want {
		t.Fatalf("call 1 = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.calls[2], " "), "start api"; got != want {
		t.Fatalf("call 2 = %q, want %q", got, want)
	}
	if !strings.Contains(out.String(), "Service api install success\n") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunIgnoresStartErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "api.service")
	write(t, file, "[Unit]\nDescription=API\n[Service]\nExecStart=/bin/api\n")

	err := Run(context.Background(), file, Options{}, Dependencies{
		Installer: &fakeInstaller{},
		Runner:    &fakeRunner{err: errors.New("boom")},
		Out:       io.Discard,
	})
	if err != nil {
		t.Fatalf("run install-service should ignore systemctl errors, got %v", err)
	}
}

func TestFSInstallerWritesClassicServiceAndSymlink(t *testing.T) {
	root := t.TempDir()
	fs := otterfs.New(otterfs.Config{
		ClassicServicePath: filepath.Join(root, "classic"),
		SystemdServicePath: filepath.Join(root, "systemd"),
	})

	err := FSInstaller{FS: fs}.Install(context.Background(), "api", []byte("[Unit]\nDescription=API\n[Service]\nExecStart=/bin/api\n"))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	target, err := os.Readlink(filepath.Join(root, "systemd", "api.service"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != filepath.Join(root, "classic", "api.service") {
		t.Fatalf("symlink target = %q", target)
	}
}

type fakeInstaller struct {
	name string
	data []byte
	err  error
}

func (f *fakeInstaller) Install(ctx context.Context, name string, data []byte) error {
	f.name = name
	f.data = append([]byte(nil), data...)
	return f.err
}

type fakeRunner struct {
	calls [][]string
	err   error
}

func (f *fakeRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	return f.err
}

func write(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
