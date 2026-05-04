package showports

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRunPrintsListeningConnections(t *testing.T) {
	var out bytes.Buffer
	err := Run(context.Background(), []string{"api.service", "worker"}, Dependencies{
		PidFinder: fakePidFinder{pids: map[string][]int32{
			"api":    {10},
			"worker": {20},
		}},
		ConnFinder: fakeConnFinder{connections: map[int32][]Connection{
			10: {
				{Type: syscall.SOCK_STREAM, Status: listenState, IP: "127.0.0.1", Port: 8080},
				{Type: syscall.SOCK_STREAM, Status: "ESTABLISHED", IP: "127.0.0.1", Port: 9000},
			},
			20: {{Type: syscall.SOCK_DGRAM, Status: listenState, IP: "0.0.0.0", Port: 53}},
		}},
		Out: &out,
	})
	if err != nil {
		t.Fatalf("run show-ports: %v", err)
	}
	want := "TCP Listen 127.0.0.1:8080\nUDP Listen 0.0.0.0:53\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunWrapsPidError(t *testing.T) {
	err := Run(context.Background(), []string{"api.service"}, Dependencies{
		PidFinder: fakePidFinder{err: errors.New("boom")},
		Out:       &bytes.Buffer{},
	})
	if err == nil || err.Error() != "find pids for service 'api' failed: boom" {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestRunSkipsConnectionErrors(t *testing.T) {
	var out bytes.Buffer
	err := Run(context.Background(), []string{"api"}, Dependencies{
		PidFinder: fakePidFinder{pids: map[string][]int32{"api": {10, 20}}},
		ConnFinder: fakeConnFinder{
			errPids:     map[int32]error{10: errors.New("missing")},
			connections: map[int32][]Connection{20: {{Type: syscall.SOCK_STREAM, Status: listenState, IP: "127.0.0.1", Port: 8080}}},
		},
		Out: &out,
	})
	if err != nil {
		t.Fatalf("run show-ports: %v", err)
	}
	if got, want := out.String(), "TCP Listen 127.0.0.1:8080\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestProcConnFinderReadsTcpListen(t *testing.T) {
	root := t.TempDir()
	fdPath := filepath.Join(root, "100", "fd")
	if err := os.MkdirAll(fdPath, 0o755); err != nil {
		t.Fatalf("mkdir fd: %v", err)
	}
	if err := os.Symlink("socket:[12345]", filepath.Join(fdPath, "3")); err != nil {
		t.Fatalf("symlink socket: %v", err)
	}
	netPath := filepath.Join(root, "net")
	if err := os.MkdirAll(netPath, 0o755); err != nil {
		t.Fatalf("mkdir net: %v", err)
	}
	tcp := strings.Join([]string{
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode",
		"   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000   100        0 12345 1 0000000000000000 100 0 0 10 0",
		"   1: 0200007F:2328 00000000:0000 01 00000000:00000000 00:00000000 00000000   100        0 12345 1 0000000000000000 100 0 0 10 0",
	}, "\n")
	if err := os.WriteFile(filepath.Join(netPath, "tcp"), []byte(tcp), 0o644); err != nil {
		t.Fatalf("write tcp: %v", err)
	}

	connections, err := ProcConnFinder{Root: root}.Find(context.Background(), 100)
	if err != nil {
		t.Fatalf("find connections: %v", err)
	}
	if len(connections) != 2 {
		t.Fatalf("connections = %#v, want 2", connections)
	}
	if got := connections[0]; got.Type != syscall.SOCK_STREAM || got.Status != listenState || got.IP != "127.0.0.1" || got.Port != 8080 {
		t.Fatalf("first connection = %#v", got)
	}
}

func TestProcConnFinderMissingProcessReturnsError(t *testing.T) {
	_, err := ProcConnFinder{Root: t.TempDir()}.Find(context.Background(), 100)
	if err == nil {
		t.Fatalf("expected missing process error")
	}
}

type fakePidFinder struct {
	pids map[string][]int32
	err  error
}

func (f fakePidFinder) Find(ctx context.Context, serviceName string) ([]int32, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pids[serviceName], nil
}

type fakeConnFinder struct {
	connections map[int32][]Connection
	errPids     map[int32]error
}

func (f fakeConnFinder) Find(ctx context.Context, pid int32) ([]Connection, error) {
	if err := f.errPids[pid]; err != nil {
		return nil, err
	}
	return f.connections[pid], nil
}
