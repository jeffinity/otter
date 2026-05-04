package linkservice

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jeffinity/otter/internal/otterfs"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

type Dependencies struct {
	Store statuscmd.Store
	FS    otterfs.Provider
	Out   io.Writer
}

func Run(ctx context.Context, serviceName string, deps Dependencies) error {
	name := normalize(serviceName)
	if name == "otter-core" {
		return fmt.Errorf("otter-core must be managed outside of otter service")
	}
	fs := deps.FS
	if fs.Config().ClassicServicePath == "" {
		fs = otterfs.Default()
	}
	target := filepath.Join(fs.ClassicServicePath(), name+".service")
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("service %s has already been installed or faked", name)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	unit, err := findUnit(ctx, name, deps)
	if err != nil {
		return fmt.Errorf("cannot find service %s in systemd: %w", name, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.Symlink(unit, target); err != nil {
		return err
	}
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	_, err = fmt.Fprintf(out, "Create fake service (linked to %s) success.\n", unit)
	return err
}

func findUnit(ctx context.Context, name string, deps Dependencies) (string, error) {
	store := deps.Store
	if store == nil {
		store = statuscmd.NewSystemdStore(nil, deps.FS)
	}
	services, err := store.List(ctx)
	if err != nil {
		return "", err
	}
	for _, service := range services {
		if service.Name == name && service.FragmentPath != "" {
			return service.FragmentPath, nil
		}
	}
	return "", fmt.Errorf("service not found")
}

func normalize(name string) string {
	if len(name) > len(".service") && name[len(name)-len(".service"):] == ".service" {
		return name[:len(name)-len(".service")]
	}
	return name
}
