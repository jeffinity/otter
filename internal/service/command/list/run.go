package list

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

type Options struct {
	OneLine         bool
	ExcludeEnabled  bool
	IncludeDisabled bool
	OnlyPackage     bool
	OnlyClassic     bool
}

type Dependencies struct {
	Store statuscmd.Store
	Out   io.Writer
}

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	services, err := statuscmd.Select(ctx, args, statuscmd.Options{
		ExcludeEnabled:  opts.ExcludeEnabled,
		IncludeDisabled: opts.IncludeDisabled,
		OnlyPackage:     opts.OnlyPackage,
		OnlyClassic:     opts.OnlyClassic,
	}, statuscmd.Dependencies{Store: deps.Store})
	if err != nil {
		return err
	}
	return render(deps.Out, services, opts)
}

func render(out io.Writer, services []statuscmd.Service, opts Options) error {
	if out == nil {
		out = os.Stdout
	}
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Name)
	}
	if opts.OneLine {
		_, err := fmt.Fprintln(out, strings.Join(names, " "))
		return err
	}
	for _, name := range names {
		if _, err := fmt.Fprintln(out, name); err != nil {
			return err
		}
	}
	return nil
}
