package showpids

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunPrintsJoinedPids(t *testing.T) {
	var out bytes.Buffer
	finder := fakeFinder{pids: map[string][]int32{
		"api":    {12, 13},
		"worker": {21},
	}}

	err := Run(context.Background(), []string{"api.service", "worker"}, Dependencies{
		Finder: finder,
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run show-pids: %v", err)
	}
	if got, want := out.String(), "12 13 21\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunWrapsFindError(t *testing.T) {
	err := Run(context.Background(), []string{"api.service"}, Dependencies{
		Finder: fakeFinder{err: errors.New("boom")},
		Out:    &bytes.Buffer{},
	})
	if err == nil || err.Error() != "find pids for service 'api' failed: boom" {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestDefaultFinderReadsCgroupV2(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cgroup.controllers"), []byte("cpu\n"), 0o644); err != nil {
		t.Fatalf("write controllers: %v", err)
	}
	writeProcs(t, filepath.Join(root, "system.slice", "api.service"), "10\n11\n")

	pids, err := DefaultFinder{Root: root}.Find(context.Background(), "api")
	if err != nil {
		t.Fatalf("find pids: %v", err)
	}
	assertPids(t, pids, []int32{10, 11})
}

func TestDefaultFinderReadsCgroupV1(t *testing.T) {
	root := t.TempDir()
	writeProcs(t, filepath.Join(root, "systemd", "system.slice", "api.service"), "20\n")

	pids, err := DefaultFinder{Root: root}.Find(context.Background(), "api")
	if err != nil {
		t.Fatalf("find pids: %v", err)
	}
	assertPids(t, pids, []int32{20})
}

func TestDefaultFinderMissingCgroupReturnsNoPids(t *testing.T) {
	pids, err := DefaultFinder{Root: t.TempDir()}.Find(context.Background(), "api")
	if err != nil {
		t.Fatalf("find pids: %v", err)
	}
	if len(pids) != 0 {
		t.Fatalf("pids = %#v, want empty", pids)
	}
}

func TestDefaultFinderInvalidPidReturnsError(t *testing.T) {
	root := t.TempDir()
	writeProcs(t, filepath.Join(root, "systemd", "system.slice", "api.service"), "abc\n")

	_, err := DefaultFinder{Root: root}.Find(context.Background(), "api")
	if err == nil || err.Error() != "invalid pid 'abc'" {
		t.Fatalf("expected invalid pid error, got %v", err)
	}
}

func writeProcs(t *testing.T, dir string, data string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir cgroup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cgroup.procs"), []byte(data), 0o644); err != nil {
		t.Fatalf("write cgroup.procs: %v", err)
	}
}

func assertPids(t *testing.T, got []int32, want []int32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("pids = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("pids = %#v, want %#v", got, want)
		}
	}
}

type fakeFinder struct {
	pids map[string][]int32
	err  error
}

func (f fakeFinder) Find(ctx context.Context, serviceName string) ([]int32, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pids[serviceName], nil
}
