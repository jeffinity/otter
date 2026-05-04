package status

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
)

type ManagedSystemdStore struct {
	runner Runner
	fs     otterfs.Provider
}

func NewManagedSystemdStore(runner Runner, fs otterfs.Provider) *ManagedSystemdStore {
	if runner == nil {
		runner = execRunner{}
	}
	if fs.Config().ClassicServicePath == "" && fs.Config().PackageServicePath == "" {
		fs = otterfs.Default()
	}
	return &ManagedSystemdStore{runner: runner, fs: fs}
}

func (s *ManagedSystemdStore) List(ctx context.Context) ([]Service, error) {
	services, units, err := s.loadManagedServices()
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, nil
	}

	showOut, err := s.runner.Run(ctx, "systemctl", systemdShowArgs(units)...)
	if err != nil {
		return nil, fmt.Errorf("systemctl show: %w", err)
	}
	for _, props := range parseShow(string(showOut)) {
		unit := props["Id"]
		service, ok := services[unit]
		if !ok {
			continue
		}
		service.UnitName = unit
		service.Name = trimUnit(unit)
		mergeProperties(&service, props, s.fs)
		services[unit] = service
	}

	return servicesToSlice(services, units), nil
}

func (s *ManagedSystemdStore) loadManagedServices() (map[string]Service, []string, error) {
	services := map[string]Service{}
	if classicPath := s.fs.ClassicServicePath(); classicPath != "" {
		if err := addManagedServices(services, filepath.Join(classicPath, "*.service"), SourceSystemd); err != nil {
			return nil, nil, err
		}
	}
	if packagePath := s.fs.PackageServicePath(); packagePath != "" {
		if err := addManagedServices(services, filepath.Join(packagePath, "*", "*", "*.service"), SourcePackage); err != nil {
			return nil, nil, err
		}
	}

	units := make([]string, 0, len(services))
	for unit := range services {
		units = append(units, unit)
	}
	sort.Strings(units)
	return services, units, nil
}

func addManagedServices(services map[string]Service, pattern string, source string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, match := range matches {
		unit := filepath.Base(match)
		if strings.HasPrefix(unit, ".") || !isConcreteServiceUnit(unit) {
			continue
		}
		if _, exists := services[unit]; exists {
			continue
		}
		services[unit] = Service{
			Name:     trimUnit(unit),
			UnitName: unit,
			Source:   source,
		}
	}
	return nil
}
