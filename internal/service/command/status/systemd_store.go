package status

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jeffinity/otter/internal/otterfs"
)

type SystemdStore struct {
	runner Runner
	fs     otterfs.Provider
}

func NewSystemdStore(runner Runner, fs otterfs.Provider) *SystemdStore {
	if runner == nil {
		runner = execRunner{}
	}
	if fs.Config().SystemdServicePath == "" {
		fs = otterfs.Default()
	}
	return &SystemdStore{runner: runner, fs: fs}
}

func (s *SystemdStore) List(ctx context.Context) ([]Service, error) {
	services, err := s.loadBaseServices(ctx)
	if err != nil {
		return nil, err
	}
	units := make([]string, 0, len(services))
	for unit := range services {
		units = append(units, unit)
	}
	sort.Strings(units)
	if len(units) == 0 {
		return nil, nil
	}

	showOut, err := s.runner.Run(ctx, "systemctl", systemdShowArgs(units)...)
	if err != nil {
		return nil, fmt.Errorf("systemctl show: %w", err)
	}
	for _, props := range parseShow(string(showOut)) {
		unit := props["Id"]
		if unit == "" {
			continue
		}
		service := services[unit]
		service.UnitName = unit
		service.Name = trimUnit(unit)
		mergeProperties(&service, props, s.fs)
		services[unit] = service
	}

	return servicesToSlice(services, units), nil
}

func (s *SystemdStore) loadBaseServices(ctx context.Context) (map[string]Service, error) {
	fileOut, err := s.runner.Run(ctx, "systemctl", "list-unit-files", "--type=service", "--no-legend", "--no-pager")
	if err != nil {
		return nil, fmt.Errorf("systemctl list-unit-files: %w", err)
	}
	unitOut, err := s.runner.Run(ctx, "systemctl", "list-units", "--type=service", "--all", "--no-legend", "--no-pager")
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units: %w", err)
	}

	services := map[string]Service{}
	for unit, enabled := range parseUnitFiles(string(fileOut)) {
		service := services[unit]
		service.UnitName = unit
		service.Name = trimUnit(unit)
		service.Enabled = enabled
		services[unit] = service
	}
	for unit, state := range parseUnits(string(unitOut)) {
		service := services[unit]
		service.UnitName = unit
		service.Name = trimUnit(unit)
		service.ActiveState = state.active
		service.SubState = state.sub
		service.Running = state.active == "active"
		services[unit] = service
	}
	return services, nil
}

func systemdShowArgs(units []string) []string {
	args := []string{
		"show",
		"--no-pager",
		"--property=Id",
		"--property=FragmentPath",
		"--property=UnitFileState",
		"--property=ActiveState",
		"--property=SubState",
		"--property=MainPID",
		"--property=ActiveEnterTimestamp",
		"--property=ActiveEnterTimestampMonotonic",
		"--property=InactiveEnterTimestamp",
		"--property=InactiveEnterTimestampMonotonic",
	}
	return append(args, units...)
}

func servicesToSlice(services map[string]Service, units []string) []Service {
	result := make([]Service, 0, len(units))
	for _, unit := range units {
		service := services[unit]
		if service.Source == "" {
			service.Source = SourceSystemd
		}
		result = append(result, service)
	}
	return result
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

type unitState struct {
	active string
	sub    string
}

func parseUnitFiles(out string) map[string]bool {
	result := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !isConcreteServiceUnit(fields[0]) {
			continue
		}
		result[fields[0]] = strings.HasPrefix(fields[1], "enabled")
	}
	return result
}

func parseUnits(out string) map[string]unitState {
	result := map[string]unitState{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "●" {
			fields = fields[1:]
		}
		if len(fields) < 4 || !isConcreteServiceUnit(fields[0]) {
			continue
		}
		result[fields[0]] = unitState{active: fields[2], sub: fields[3]}
	}
	return result
}

func isConcreteServiceUnit(unit string) bool {
	return strings.HasSuffix(unit, ".service") && !strings.HasSuffix(unit, "@.service")
}

func parseShow(out string) []map[string]string {
	var records []map[string]string
	current := map[string]string{}
	flush := func() {
		if len(current) == 0 {
			return
		}
		records = append(records, current)
		current = map[string]string{}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if key == "Id" && current["Id"] != "" {
			flush()
		}
		current[key] = value
	}
	flush()
	return records
}

func mergeProperties(service *Service, props map[string]string, fs otterfs.Provider) {
	service.FragmentPath = props["FragmentPath"]
	if state := props["UnitFileState"]; state != "" {
		service.Enabled = strings.HasPrefix(state, "enabled")
	}
	if active := props["ActiveState"]; active != "" {
		service.ActiveState = active
		service.Running = active == "active"
	}
	if sub := props["SubState"]; sub != "" {
		service.SubState = sub
	}
	if pid, err := strconv.Atoi(props["MainPID"]); err == nil {
		service.MainPID = pid
	}
	service.ActiveTime = parseSystemdTime(props["ActiveEnterTimestamp"])
	service.InactiveTime = parseSystemdTime(props["InactiveEnterTimestamp"])
	service.ActiveTimeMono = parseInt64(props["ActiveEnterTimestampMonotonic"])
	service.InactiveTimeMono = parseInt64(props["InactiveEnterTimestampMonotonic"])
	if service.Source == SourcePackage || isPackagePath(service.FragmentPath, fs.PackageServicePath()) {
		service.Source = SourcePackage
	} else {
		service.Source = SourceSystemd
	}
}

func trimUnit(unit string) string {
	return strings.TrimSuffix(unit, ".service")
}

func parseInt64(value string) int64 {
	if value == "" || value == "0" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed * int64(time.Microsecond)
}

func parseSystemdTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" || value == "n/a" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"Mon 2006-01-02 15:04:05 MST",
		"Mon 2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func isPackagePath(fragmentPath, packagePath string) bool {
	fragmentPath = strings.TrimRight(fragmentPath, "/")
	packagePath = strings.TrimRight(packagePath, "/")
	return packagePath != "" && (fragmentPath == packagePath || strings.HasPrefix(fragmentPath, packagePath+"/"))
}
