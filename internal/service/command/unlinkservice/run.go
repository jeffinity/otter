package unlinkservice

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jeffinity/otter/internal/otterfs"
)

type Dependencies struct {
	FS  otterfs.Provider
	Out io.Writer
}

func Run(ctx context.Context, serviceName string, deps Dependencies) error {
	_ = ctx

	name := normalize(serviceName)
	fs := deps.FS
	if fs.Config().ClassicServicePath == "" {
		fs = otterfs.Default()
	}

	target := filepath.Join(fs.ClassicServicePath(), name+".service")
	info, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service %s is not linked by otter", name)
		}
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("service %s is managed by otter but not linked", name)
	}
	if err := os.Remove(target); err != nil {
		return err
	}

	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	_, err = fmt.Fprintf(out, "Remove fake service link %s success.\n", target)
	return err
}

func normalize(name string) string {
	if len(name) > len(".service") && name[len(name)-len(".service"):] == ".service" {
		return name[:len(name)-len(".service")]
	}
	return name
}
