package daemonreload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestRunExecutesDaemonReload(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeRunner{}

	err := Run(context.Background(), Dependencies{
		Runner: runner,
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run daemon-reload: %v", err)
	}
	if got, want := out.String(), "systemctl daemon-reload \n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if got, want := runner.args, []string{"daemon-reload"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestRunReturnsRunnerError(t *testing.T) {
	runner := &fakeRunner{err: errors.New("boom")}

	err := Run(context.Background(), Dependencies{
		Runner: runner,
		Out:    io.Discard,
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected runner error, got %v", err)
	}
}

type fakeRunner struct {
	args []string
	err  error
}

func (f *fakeRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.args = append([]string(nil), args...)
	return f.err
}

var _ Runner = (*fakeRunner)(nil)
