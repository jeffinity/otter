package disable

import (
	"context"
	"io"
	"reflect"
	"testing"

	startcmd "github.com/jeffinity/otter/internal/service/command/start"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

func TestRunDisablesServices(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"api.service"}, Options{}, Dependencies{
		Store:  fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Runner: runner,
		Out:    io.Discard,
	})
	if err != nil {
		t.Fatalf("run disable: %v", err)
	}
	if got, want := runner.calls, [][]string{{"disable", "api"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
}

func TestRunStopsAfterDisable(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"all"}, Options{Stop: true}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{
			{Name: "api", Enabled: true},
			{Name: "disabled", Enabled: false},
		}},
		Runner: runner,
		Out:    io.Discard,
	})
	if err != nil {
		t.Fatalf("run disable --stop: %v", err)
	}
	want := [][]string{{"disable", "api"}, {"stop", "api"}}
	if got := runner.calls; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
}

func TestRunNoEnabled(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"all"}, Options{ExcludeEnabled: true}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{
			{Name: "api", Enabled: true},
			{Name: "disabled", Enabled: false},
		}},
		Runner: runner,
		Out:    io.Discard,
	})
	if err != nil {
		t.Fatalf("run disable --no-enabled: %v", err)
	}
	if got, want := runner.calls, [][]string{{"disable", "disabled"}}; !reflect.DeepEqual(got, want) {
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
