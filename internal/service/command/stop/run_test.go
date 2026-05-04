package stop

import (
	"context"
	"io"
	"reflect"
	"testing"

	startcmd "github.com/jeffinity/otter/internal/service/command/start"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

func TestRunStopsServices(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"api"}, Options{}, Dependencies{
		Store:  fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Runner: runner,
		Out:    io.Discard,
	})
	if err != nil {
		t.Fatalf("run stop: %v", err)
	}
	if got, want := runner.calls, [][]string{{"stop", "api"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
}

type fakeStore struct {
	services []statuscmd.Service
}

func (f fakeStore) List(ctx context.Context) ([]statuscmd.Service, error) {
	return f.services, nil
}

type fakeRunner struct {
	calls [][]string
}

func (f *fakeRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	return nil
}

var _ startcmd.Runner = (*fakeRunner)(nil)
