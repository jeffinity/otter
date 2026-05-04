package grouplist

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
	"github.com/jeffinity/otter/internal/service/command/servicefile"
)

type Options struct {
	OneLine         bool
	IncludeServices bool
}

type Dependencies struct {
	Store Store
	FS    otterfs.Provider
	Out   io.Writer
}

type Store interface {
	List(ctx context.Context) (map[string][]string, error)
}

func Run(ctx context.Context, opts Options, deps Dependencies) error {
	groups, err := store(deps).List(ctx)
	if err != nil {
		return err
	}
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}

	names := groupNames(groups)
	switch {
	case opts.IncludeServices:
		for _, name := range names {
			services := append([]string(nil), groups[name]...)
			sort.Strings(services)
			if _, err := fmt.Fprintln(out, name+": "+strings.Join(services, ", ")); err != nil {
				return err
			}
		}
	case opts.OneLine:
		_, err = fmt.Fprintln(out, strings.Join(names, " "))
	default:
		for _, name := range names {
			if _, err := fmt.Fprintln(out, name); err != nil {
				return err
			}
		}
	}
	return err
}

func groupNames(groups map[string][]string) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func store(deps Dependencies) Store {
	if deps.Store != nil {
		return deps.Store
	}
	fs := deps.FS
	if fs.Config().ClassicServicePath == "" {
		fs = otterfs.Default()
	}
	return FSStore{FS: fs}
}

type FSStore struct {
	FS otterfs.Provider
}

func (s FSStore) List(ctx context.Context) (map[string][]string, error) {
	_ = ctx

	groups := map[string][]string{}
	for _, pattern := range []string{
		filepath.Join(s.FS.ClassicServicePath(), "*.service"),
		filepath.Join(s.FS.PackageServicePath(), "*", "*", "*.service"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			serviceName := strings.TrimSuffix(filepath.Base(match), ".service")
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			for _, group := range parseGroups(string(data)) {
				groups[group] = append(groups[group], serviceName)
			}
		}
	}
	return groups, nil
}

func parseGroups(data string) []string {
	values, exists, err := servicefile.Values(data, "Group")
	if err != nil || !exists {
		return nil
	}
	groups := make([]string, 0)
	for _, value := range values {
		for _, group := range strings.Split(value, ",") {
			group = strings.TrimSpace(group)
			if group != "" {
				groups = append(groups, group)
			}
		}
	}
	return groups
}
