package showpids

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Dependencies struct {
	Finder Finder
	Out    io.Writer
}

type Finder interface {
	Find(ctx context.Context, serviceName string) ([]int32, error)
}

func Run(ctx context.Context, args []string, deps Dependencies) error {
	finder := deps.Finder
	if finder == nil {
		finder = DefaultFinder{}
	}

	pids := make([]int32, 0)
	for _, arg := range args {
		serviceName := strings.TrimSuffix(arg, ".service")
		servicePids, err := finder.Find(ctx, serviceName)
		if err != nil {
			return fmt.Errorf("find pids for service '%s' failed: %w", serviceName, err)
		}
		pids = append(pids, servicePids...)
	}

	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	_, err := fmt.Fprintln(out, joinPids(pids, " "))
	return err
}

func joinPids(pids []int32, sep string) string {
	parts := make([]string, 0, len(pids))
	for _, pid := range pids {
		parts = append(parts, strconv.Itoa(int(pid)))
	}
	return strings.Join(parts, sep)
}

type DefaultFinder struct {
	Root string
	V2   *bool
}

func (f DefaultFinder) Find(ctx context.Context, serviceName string) ([]int32, error) {
	_ = ctx

	root := f.Root
	if root == "" {
		root = "/sys/fs/cgroup"
	}
	path := filepath.Join(root, "systemd", "system.slice", serviceName+".service")
	if f.isV2(root) {
		path = filepath.Join(root, "system.slice", serviceName+".service")
	}
	return readCgroupPids(path)
}

func (f DefaultFinder) isV2(root string) bool {
	if f.V2 != nil {
		return *f.V2
	}
	_, err := os.Stat(filepath.Join(root, "cgroup.controllers"))
	return err == nil
}

func readCgroupPids(path string) ([]int32, error) {
	if stat, err := os.Stat(path); err != nil || !stat.IsDir() {
		return nil, nil
	}

	out, err := os.ReadFile(filepath.Join(path, "cgroup.procs"))
	if err != nil {
		return nil, err
	}

	pids := make([]int32, 0)
	for _, pidBS := range bytes.Split(out, []byte{'\n'}) {
		if len(pidBS) == 0 {
			continue
		}
		pid, err := strconv.ParseInt(string(pidBS), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid pid '%s'", pidBS)
		}
		pids = append(pids, int32(pid))
	}
	return pids, nil
}
