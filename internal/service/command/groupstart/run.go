package groupstart

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jeffinity/otter/internal/otterfs"
	grouplistcmd "github.com/jeffinity/otter/internal/service/command/grouplist"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

type Options struct {
	StopAfter time.Duration
}

type Dependencies struct {
	GroupStore  grouplistcmd.Store
	FS          otterfs.Provider
	ActionStore statuscmd.Store
	Runner      startcmd.Runner
	AutoStopper startcmd.AutoStopper
	TraceRunner startcmd.TraceRunner
	Executable  func() (string, error)
	Environ     func() []string
	Out         io.Writer
	ErrOut      io.Writer
	In          io.Reader
}

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	return RunAction(ctx, "start", args, opts, deps)
}

func RunAction(ctx context.Context, action string, args []string, opts Options, deps Dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one group should be provided")
	}

	services, err := servicesForGroups(ctx, args, deps)
	if err != nil {
		return err
	}

	return startcmd.RunAction(ctx, action, services, startcmd.Options{
		StopAfter: opts.StopAfter,
	}, actionDeps(deps))
}

func servicesForGroups(ctx context.Context, args []string, deps Dependencies) ([]string, error) {
	groups, err := groupStore(deps).List(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get services from group: %w", err)
	}

	services := make([]string, 0)
	for _, group := range args {
		items, ok := groups[group]
		if !ok {
			return nil, fmt.Errorf("cannot get services from group: group %s is not exist", group)
		}
		services = append(services, items...)
	}
	return dedupe(services), nil
}

func dedupe(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func groupStore(deps Dependencies) grouplistcmd.Store {
	if deps.GroupStore != nil {
		return deps.GroupStore
	}
	fs := deps.FS
	if fs.Config().ClassicServicePath == "" {
		fs = otterfs.Default()
	}
	return grouplistcmd.FSStore{FS: fs}
}

func actionDeps(deps Dependencies) startcmd.Dependencies {
	store := deps.ActionStore
	if store == nil {
		store = statuscmd.NewManagedSystemdStore(nil, deps.FS)
	}
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := deps.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}
	in := deps.In
	if in == nil {
		in = os.Stdin
	}
	return startcmd.Dependencies{
		Store:       store,
		Runner:      deps.Runner,
		AutoStopper: deps.AutoStopper,
		TraceRunner: deps.TraceRunner,
		Executable:  deps.Executable,
		Environ:     deps.Environ,
		Out:         out,
		ErrOut:      errOut,
		In:          in,
	}
}
