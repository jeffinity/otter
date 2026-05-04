package log

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jeffinity/otter/internal/service/command/servicefile"
)

func TestRunUsesJournalctlByDefault(t *testing.T) {
	servicePath := writeService(t, "[Unit]\nDescription=API\n")
	runner := &fakeRunner{}

	out, err := runLog(t, "api.service", Options{}, fakeFinder{file: fileFor(servicePath)}, runner, nil)
	if err != nil {
		t.Fatalf("run log: %v", err)
	}
	if got, want := runner.command, "/bin/journalctl"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
	wantArgs := []string{"journalctl", "--unit=api", "--output=cat", "--lines=80"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", runner.args, wantArgs)
	}
	if got, want := out, "journalctl --unit=api --output=cat --lines=80\n"; got != want {
		t.Fatalf("preview = %q, want %q", got, want)
	}
}

func TestRunUsesJournalctlOptions(t *testing.T) {
	servicePath := writeService(t, "[Unit]\n")
	runner := &fakeRunner{}
	now := func() time.Time {
		return time.Date(2026, 4, 28, 12, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	}

	_, err := runLog(t, "api", Options{
		Follow:   true,
		Lines:    -1,
		Since:    "2h",
		Until:    "2026-04-28 12:30",
		Output:   "short",
		PagerEnd: true,
		Reverse:  true,
	}, fakeFinder{file: fileFor(servicePath)}, runner, now)
	if err != nil {
		t.Fatalf("run log: %v", err)
	}
	wantArgs := []string{
		"journalctl",
		"--unit=api",
		"--output=short",
		"--lines=all",
		"--follow",
		"--pager-end",
		"--reverse",
		"--since=2026-04-28 10:00:00",
		"--until=2026-04-28 12:30:00",
	}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", runner.args, wantArgs)
	}
}

func TestRunUsesCustomLogWithLess(t *testing.T) {
	servicePath := writeService(t, "[X-Otter]\nLogFile=/var/log/api.log\n")
	runner := &fakeRunner{}

	_, err := runLog(t, "api", Options{}, fakeFinder{file: fileFor(servicePath)}, runner, nil)
	if err != nil {
		t.Fatalf("run log: %v", err)
	}
	if got, want := runner.command, "/bin/bash"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
	wantArgs := []string{"bash", "-c", "tail --lines=80 /var/log/api.log 2>&1 | '/bin/less'"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", runner.args, wantArgs)
	}
}

func TestRunUsesCustomLogFollow(t *testing.T) {
	servicePath := writeService(t, "[X-Otter]\nLogFile=/var/log/api.log\n")
	runner := &fakeRunner{}

	_, err := runLog(t, "api", Options{Follow: true, Lines: 20}, fakeFinder{file: fileFor(servicePath)}, runner, nil)
	if err != nil {
		t.Fatalf("run log: %v", err)
	}
	wantArgs := []string{"tail", "--lines=20", "-F", "/var/log/api.log"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", runner.args, wantArgs)
	}
}

func TestRunForceJournalctlIgnoresCustomLog(t *testing.T) {
	servicePath := writeService(t, "[X-Otter]\nLogFile=/var/log/api.log\n")
	runner := &fakeRunner{}

	_, err := runLog(t, "api", Options{ForceJournalctl: true}, fakeFinder{file: fileFor(servicePath)}, runner, nil)
	if err != nil {
		t.Fatalf("run log: %v", err)
	}
	if got, want := runner.command, "/bin/journalctl"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestRunReturnsLookPathError(t *testing.T) {
	servicePath := writeService(t, "[Unit]\n")
	_, err := runLogWithLookPath(t, "api", Options{}, fakeFinder{file: fileFor(servicePath)}, &fakeRunner{}, nil, func(file string) (string, error) {
		return "", errors.New("missing")
	})
	if err == nil || !strings.Contains(err.Error(), "cannot found journalctl in PATH") {
		t.Fatalf("expected journalctl lookup error, got %v", err)
	}
}

func TestRunReturnsRunnerError(t *testing.T) {
	servicePath := writeService(t, "[Unit]\n")
	runner := &fakeRunner{err: errors.New("boom")}

	_, err := runLog(t, "api", Options{}, fakeFinder{file: fileFor(servicePath)}, runner, nil)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected runner error, got %v", err)
	}
}

func TestRunReturnsFindError(t *testing.T) {
	_, err := runLog(t, "missing", Options{}, fakeFinder{err: os.ErrNotExist}, &fakeRunner{}, nil)
	if err == nil {
		t.Fatalf("expected find error")
	}
}

func runLog(
	t *testing.T,
	serviceName string,
	opts Options,
	finder servicefile.Finder,
	runner *fakeRunner,
	now func() time.Time,
) (string, error) {
	t.Helper()
	return runLogWithLookPath(t, serviceName, opts, finder, runner, now, fakeLookPath)
}

func runLogWithLookPath(
	t *testing.T,
	serviceName string,
	opts Options,
	finder servicefile.Finder,
	runner *fakeRunner,
	now func() time.Time,
	lookPath func(string) (string, error),
) (string, error) {
	t.Helper()
	var out bytes.Buffer
	opts.NoColor = true
	err := Run(context.Background(), serviceName, opts, Dependencies{
		Finder:   finder,
		Runner:   runner,
		LookPath: lookPath,
		Out:      &out,
		ErrOut:   io.Discard,
		Now:      now,
	})
	return out.String(), err
}

func fakeLookPath(file string) (string, error) {
	paths := map[string]string{
		"journalctl": "/bin/journalctl",
		"tail":       "/bin/tail",
		"less":       "/bin/less",
		"bash":       "/bin/bash",
	}
	if p, ok := paths[file]; ok {
		return p, nil
	}
	return "", errors.New("missing")
}

func writeService(t *testing.T, data string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "api.service")
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatalf("write service: %v", err)
	}
	return p
}

func fileFor(path string) servicefile.File {
	return servicefile.File{Name: "api", Path: path, Source: servicefile.SourceClassic}
}

type fakeRunner struct {
	command string
	args    []string
	err     error
}

func (f *fakeRunner) Run(ctx context.Context, command string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.command = command
	f.args = append([]string(nil), args...)
	return f.err
}

type fakeFinder struct {
	file servicefile.File
	err  error
}

func (f fakeFinder) Find(ctx context.Context, serviceName string) (servicefile.File, error) {
	return f.file, f.err
}
