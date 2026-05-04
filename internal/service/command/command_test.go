package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffinity/otter/internal/otterfs"
	auditcmd "github.com/jeffinity/otter/internal/service/command/audit"
	daemonreloadcmd "github.com/jeffinity/otter/internal/service/command/daemonreload"
	detailcmd "github.com/jeffinity/otter/internal/service/command/detail"
	editcmd "github.com/jeffinity/otter/internal/service/command/edit"
	grouplistcmd "github.com/jeffinity/otter/internal/service/command/grouplist"
	installcmd "github.com/jeffinity/otter/internal/service/command/install"
	installservicecmd "github.com/jeffinity/otter/internal/service/command/installservice"
	logcmd "github.com/jeffinity/otter/internal/service/command/log"
	regeneratecmd "github.com/jeffinity/otter/internal/service/command/regenerate"
	selfcheckcmd "github.com/jeffinity/otter/internal/service/command/selfcheck"
	"github.com/jeffinity/otter/internal/service/command/servicefile"
	showpidscmd "github.com/jeffinity/otter/internal/service/command/showpids"
	showportscmd "github.com/jeffinity/otter/internal/service/command/showports"
	showpropertycmd "github.com/jeffinity/otter/internal/service/command/showproperty"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
	upsertclustercmd "github.com/jeffinity/otter/internal/service/command/upsertcluster"
	upsertselfcmd "github.com/jeffinity/otter/internal/service/command/upsertself"
)

func TestServiceCommandUnsupportedPlatform(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "darwin"})
	cmd.SetArgs([]string{"status"})

	err := cmd.Execute()
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func TestServiceCommandDefaultsToStatus(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		StatusStore: fakeStatusStore{
			services: []statuscmd.Service{{Name: "api", UnitName: "api.service", Enabled: true, ActiveState: "active", SubState: "running", Running: true}},
		},
		Out: &out,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected default status dispatch, got %v", err)
	}
	if !strings.Contains(out.String(), "api") || !strings.Contains(out.String(), "running") {
		t.Fatalf("expected status output, got %q", out.String())
	}
}

func TestServiceCommandTree(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})

	if flag := cmd.PersistentFlags().Lookup("cluster"); flag != nil {
		t.Fatalf("service command should not expose cluster flag")
	}
	if flag := cmd.PersistentFlags().ShorthandLookup("c"); flag != nil {
		t.Fatalf("service command should not expose -c shorthand")
	}

	for _, tc := range []struct {
		name    string
		aliases []string
		hidden  bool
	}{
		{name: "status"},
		{name: "list"},
		{name: "detail"},
		{name: "show-property", aliases: []string{"show"}},
		{name: "view"},
		{name: "log", aliases: []string{"logs"}},
		{name: "show-pids", aliases: []string{"show-pid", "pids", "pid"}},
		{name: "show-ports", aliases: []string{"show-port", "ports", "port"}},
		{name: "start"},
		{name: "stop"},
		{name: "restart"},
		{name: "reload"},
		{name: "enable"},
		{name: "disable"},
		{name: "daemon-reload"},
		{name: "group-list", aliases: []string{"list-group"}},
		{name: "group-start", aliases: []string{"start-group"}},
		{name: "group-stop", aliases: []string{"stop-group"}},
		{name: "group-restart", aliases: []string{"restart-group"}},
		{name: "install-service", aliases: []string{"iiiii"}},
		{name: "install-command"},
		{name: "exp-install-docker-compose", aliases: []string{"install-docker-compose", "idc"}},
		{name: "link-service", aliases: []string{"link", "install-fake", "fake", "install-fake-service"}},
		{name: "edit"},
		{name: "re-generate", aliases: []string{"regen"}, hidden: true},
		{name: "audit", hidden: true},
		{name: "self-check", aliases: []string{"check"}, hidden: true},
		{name: "install", hidden: true},
		{name: "upsert-self", aliases: []string{"install-self", "self-install", "self-update", "update-self", "us"}, hidden: true},
		{name: "upsert-cluster", aliases: []string{"uc", "i-c", "ii-c", "i-cluster", "install-cluster", "update-cluster"}, hidden: true},
	} {
		child := assertCommand(t, cmd, tc.name)
		if child.Hidden != tc.hidden {
			t.Fatalf("command %q hidden = %v, want %v", tc.name, child.Hidden, tc.hidden)
		}
		for _, alias := range tc.aliases {
			found, _, err := cmd.Find([]string{alias})
			if err != nil {
				t.Fatalf("find alias %q: %v", alias, err)
			}
			if found == nil || found.Name() != tc.name {
				t.Fatalf("alias %q resolved to %v, want %q", alias, commandName(found), tc.name)
			}
		}
	}
}

func TestServiceCommandFlags(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})

	for _, name := range []string{"status", "list"} {
		child := assertCommand(t, cmd, name)
		assertFlags(t, child, "no-enabled", "disabled", "only-package", "only-classic")
	}

	status := assertCommand(t, cmd, "status")
	assertFlags(t, status, "time-info", "asc", "desc", "since", "no-mono")

	logCmd := assertCommand(t, cmd, "log")
	assertFlags(t, logCmd, "follow", "lines", "since", "until", "output", "pager-end", "reverse", "force-journalctl")
	if f := logCmd.Flags().Lookup("force-journalctl"); f == nil || !f.Hidden {
		t.Fatalf("force-journalctl should be hidden")
	}

	start := assertCommand(t, cmd, "start")
	assertFlags(t, start, "no-enabled", "disabled", "reload", "stop-after", "trace")

	restart := assertCommand(t, cmd, "restart")
	assertFlags(t, restart, "no-enabled", "disabled", "reload", "trace")

	enable := assertCommand(t, cmd, "enable")
	assertFlags(t, enable, "no-enabled", "disabled", "start")

	disable := assertCommand(t, cmd, "disable")
	assertFlags(t, disable, "no-enabled", "disabled", "stop")

	detail := assertCommand(t, cmd, "detail")
	assertFlags(t, detail, "no-enabled", "disabled", "no-pager")

	installService := assertCommand(t, cmd, "install-service")
	assertFlags(t, installService, "name", "no-enable", "no-start")

	installCommand := assertCommand(t, cmd, "install-command")
	assertFlags(t, installCommand, "name", "no-enable", "no-start", "wd", "no-install")

	installDockerCompose := assertCommand(t, cmd, "exp-install-docker-compose")
	assertFlags(t, installDockerCompose, "name", "no-enable", "no-start", "dir", "force")

	regen := assertCommand(t, cmd, "re-generate")
	assertFlags(t, regen, "restart", "not-restart")

	groupStart := assertCommand(t, cmd, "group-start")
	assertFlags(t, groupStart, "stop-after")
}

func TestStatusRejectsConflictingSortFlags(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"status", "--asc", "--desc"})

	err := cmd.Execute()
	if err == nil || err.Error() != "--asc and --desc cannot be apply in the meantime" {
		t.Fatalf("expected conflicting sort flags error, got %v", err)
	}
}

func TestListCommandRuns(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		ListStore: fakeStatusStore{
			services: []statuscmd.Service{
				{Name: "worker", UnitName: "worker.service", Enabled: false},
				{Name: "api", UnitName: "api.service", Enabled: true},
			},
		},
		Out: &out,
	})
	cmd.SetArgs([]string{"list", "--disabled"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected list to run, got %v", err)
	}
	if got, want := out.String(), "api\nworker\n"; got != want {
		t.Fatalf("list output = %q, want %q", got, want)
	}
}

func TestListRejectsConflictingSourceFlags(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"list", "--only-package", "--only-classic"})

	err := cmd.Execute()
	if err == nil || err.Error() != "--only-package and --only-classic cannot be apply in the meantime" {
		t.Fatalf("expected conflicting source flags error, got %v", err)
	}
}

func TestDetailCommandRuns(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeDetailRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		DetailStore: fakeStatusStore{
			services: []statuscmd.Service{
				{Name: "worker", UnitName: "worker.service", Enabled: false},
				{Name: "api", UnitName: "api.service", Enabled: true},
			},
		},
		DetailRunner: runner,
		Out:          &out,
	})
	cmd.SetArgs([]string{"detail", "--disabled", "--no-pager", "all"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected detail to run, got %v", err)
	}
	if got, want := out.String(), "systemctl status api worker\n"; got != want {
		t.Fatalf("detail output = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.args, " "), "status api worker --no-pager"; got != want {
		t.Fatalf("detail args = %q, want %q", got, want)
	}
}

func TestShowPropertyCommandRuns(t *testing.T) {
	var out bytes.Buffer
	getter := &fakeShowPropertyGetter{properties: showpropertycmd.Properties{"LoadState": "loaded"}}
	cmd := New(Dependencies{
		RuntimeOS:          "linux",
		ShowPropertyGetter: getter,
		Out:                &out,
	})
	cmd.SetArgs([]string{"show-property", "api.service"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected show-property to run, got %v", err)
	}
	if getter.serviceName != "api" {
		t.Fatalf("service name = %q, want %q", getter.serviceName, "api")
	}
	if !strings.Contains(out.String(), "LoadState: loaded\n") {
		t.Fatalf("show-property output = %q", out.String())
	}
}

func TestShowPropertyAliasRuns(t *testing.T) {
	var out bytes.Buffer
	getter := &fakeShowPropertyGetter{properties: showpropertycmd.Properties{"LoadState": "loaded", "Zeta": "z"}}
	cmd := New(Dependencies{
		RuntimeOS:          "linux",
		ShowPropertyGetter: getter,
		Out:                &out,
	})
	cmd.SetArgs([]string{"show", "--all", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected show alias to run, got %v", err)
	}
	if !strings.Contains(out.String(), "Zeta: z\n") {
		t.Fatalf("show-property --all output = %q", out.String())
	}
}

func TestShowPropertyRequiresOneArg(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"show-property"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected exactly one arg error")
	}
}

func TestViewCommandRuns(t *testing.T) {
	var out bytes.Buffer
	servicePath := writeCommandTestFile(t, "api.service", "[Unit]\nDescription=API\n")
	cmd := New(Dependencies{
		RuntimeOS:  "linux",
		ViewFinder: fakeServiceFinder{file: servicefile.File{Name: "api", Path: servicePath, Source: servicefile.SourceClassic}},
		Out:        &out,
	})
	cmd.SetArgs([]string{"view", "api.service"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected view to run, got %v", err)
	}
	if got, want := out.String(), "[Unit]\nDescription=API\n"; got != want {
		t.Fatalf("view output = %q, want %q", got, want)
	}
}

func TestViewRequiresOneArg(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"view"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected exactly one arg error")
	}
}

func TestLogCommandRuns(t *testing.T) {
	var out bytes.Buffer
	servicePath := writeCommandTestFile(t, "api.service", "[Unit]\nDescription=API\n")
	runner := &fakeLogRunner{}
	cmd := New(Dependencies{
		RuntimeOS:   "linux",
		LogFinder:   fakeServiceFinder{file: servicefile.File{Name: "api", Path: servicePath, Source: servicefile.SourceClassic}},
		LogRunner:   runner,
		LogLookPath: commandTestLookPath,
		Out:         &out,
	})
	cmd.SetArgs([]string{"log", "api.service"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected log to run, got %v", err)
	}
	if got, want := strings.Join(runner.args, " "), "journalctl --unit=api --output=cat --lines=80"; got != want {
		t.Fatalf("log args = %q, want %q", got, want)
	}
	if got, want := out.String(), "journalctl --unit=api --output=cat --lines=80\n"; got != want {
		t.Fatalf("log output = %q, want %q", got, want)
	}
}

func TestLogAliasRuns(t *testing.T) {
	servicePath := writeCommandTestFile(t, "api.service", "[Unit]\nDescription=API\n")
	runner := &fakeLogRunner{}
	cmd := New(Dependencies{
		RuntimeOS:   "linux",
		LogFinder:   fakeServiceFinder{file: servicefile.File{Name: "api", Path: servicePath, Source: servicefile.SourceClassic}},
		LogRunner:   runner,
		LogLookPath: commandTestLookPath,
		Out:         io.Discard,
	})
	cmd.SetArgs([]string{"logs", "--lines", "20", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected logs alias to run, got %v", err)
	}
	if got, want := strings.Join(runner.args, " "), "journalctl --unit=api --output=cat --lines=20"; got != want {
		t.Fatalf("log alias args = %q, want %q", got, want)
	}
}

func TestLogRequiresOneArg(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"log"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected exactly one arg error")
	}
}

func TestDaemonReloadCommandRuns(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeDaemonReloadRunner{}
	cmd := New(Dependencies{
		RuntimeOS:          "linux",
		DaemonReloadRunner: runner,
		Out:                &out,
	})
	cmd.SetArgs([]string{"daemon-reload"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected daemon-reload to run, got %v", err)
	}
	if got, want := strings.Join(runner.args, " "), "daemon-reload"; got != want {
		t.Fatalf("daemon-reload args = %q, want %q", got, want)
	}
	if got, want := out.String(), "systemctl daemon-reload \n"; got != want {
		t.Fatalf("daemon-reload output = %q, want %q", got, want)
	}
}

func TestDaemonReloadRequiresNoArgs(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"daemon-reload", "api"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected no args error")
	}
}

func TestShowPidsCommandRuns(t *testing.T) {
	var out bytes.Buffer
	finder := fakeShowPidsFinder{pids: map[string][]int32{
		"api":    {10, 11},
		"worker": {20},
	}}
	cmd := New(Dependencies{
		RuntimeOS:      "linux",
		ShowPidsFinder: finder,
		Out:            &out,
	})
	cmd.SetArgs([]string{"show-pids", "api.service", "worker"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected show-pids to run, got %v", err)
	}
	if got, want := out.String(), "10 11 20\n"; got != want {
		t.Fatalf("show-pids output = %q, want %q", got, want)
	}
}

func TestShowPidsAliasRuns(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS:      "linux",
		ShowPidsFinder: fakeShowPidsFinder{pids: map[string][]int32{"api": {10}}},
		Out:            &out,
	})
	cmd.SetArgs([]string{"pid", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected pid alias to run, got %v", err)
	}
	if got, want := out.String(), "10\n"; got != want {
		t.Fatalf("pid output = %q, want %q", got, want)
	}
}

func TestShowPidsRequiresArgs(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"show-pids"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected minimum args error")
	}
}

func TestShowPortsCommandRuns(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS:          "linux",
		ShowPortsPidFinder: fakeShowPidsFinder{pids: map[string][]int32{"api": {10}}},
		ShowPortsConnFinder: fakeShowPortsConnFinder{connections: map[int32][]showportscmd.Connection{
			10: {{Type: syscall.SOCK_STREAM, Status: "LISTEN", IP: "127.0.0.1", Port: 8080}},
		}},
		Out: &out,
	})
	cmd.SetArgs([]string{"show-ports", "api.service"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected show-ports to run, got %v", err)
	}
	if got, want := out.String(), "TCP Listen 127.0.0.1:8080\n"; got != want {
		t.Fatalf("show-ports output = %q, want %q", got, want)
	}
}

func TestShowPortsAliasRuns(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS:          "linux",
		ShowPortsPidFinder: fakeShowPidsFinder{pids: map[string][]int32{"api": {10}}},
		ShowPortsConnFinder: fakeShowPortsConnFinder{connections: map[int32][]showportscmd.Connection{
			10: {{Type: syscall.SOCK_STREAM, Status: "LISTEN", IP: "0.0.0.0", Port: 80}},
		}},
		Out: &out,
	})
	cmd.SetArgs([]string{"port", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected port alias to run, got %v", err)
	}
	if got, want := out.String(), "TCP Listen 0.0.0.0:80\n"; got != want {
		t.Fatalf("port output = %q, want %q", got, want)
	}
}

func TestShowPortsRequiresArgs(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"show-ports"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected minimum args error")
	}
}

func TestStartCommandRuns(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		ActionStore: fakeStatusStore{
			services: []statuscmd.Service{
				{Name: "worker", Enabled: false},
				{Name: "api", Enabled: true},
			},
		},
		ActionRunner: runner,
		Out:          &out,
	})
	cmd.SetArgs([]string{"start", "--reload", "all"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected start to run, got %v", err)
	}
	assertActionCall(t, runner, 0, "daemon-reload")
	assertActionCall(t, runner, 1, "start api")
	if got, want := out.String(), "systemctl daemon-reload\nsystemctl start api\n"; got != want {
		t.Fatalf("start output = %q, want %q", got, want)
	}
}

func TestStopCommandRuns(t *testing.T) {
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		ActionStore: fakeStatusStore{
			services: []statuscmd.Service{{Name: "worker", Enabled: false}},
		},
		ActionRunner: runner,
		Out:          io.Discard,
	})
	cmd.SetArgs([]string{"stop", "worker.service"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected stop to run, got %v", err)
	}
	assertActionCall(t, runner, 0, "stop worker")
}

func TestRestartCommandRuns(t *testing.T) {
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		ActionStore: fakeStatusStore{
			services: []statuscmd.Service{{Name: "api", Enabled: true}},
		},
		ActionRunner: runner,
		Out:          io.Discard,
	})
	cmd.SetArgs([]string{"restart", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected restart to run, got %v", err)
	}
	assertActionCall(t, runner, 0, "restart api")
}

func TestReloadCommandRuns(t *testing.T) {
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		ActionStore: fakeStatusStore{
			services: []statuscmd.Service{{Name: "api", Enabled: true}},
		},
		ActionRunner: runner,
		Out:          io.Discard,
	})
	cmd.SetArgs([]string{"reload", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected reload to run, got %v", err)
	}
	assertActionCall(t, runner, 0, "reload api")
}

func TestReloadRequiresArgs(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"reload"})

	err := cmd.Execute()
	if err == nil ||
		err.Error() != "at least one service should be provided, if you want to reload service file please use `otter service daemon-reload`" {
		t.Fatalf("expected reload args error, got %v", err)
	}
}

func TestEnableCommandRuns(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		ActionStore: fakeStatusStore{
			services: []statuscmd.Service{
				{Name: "api", Enabled: true},
				{Name: "disabled", Enabled: false},
			},
		},
		ActionRunner: runner,
		Out:          &out,
	})
	cmd.SetArgs([]string{"enable", "--start", "all"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected enable to run, got %v", err)
	}
	assertActionCall(t, runner, 0, "enable api")
	assertActionCall(t, runner, 1, "start api")
	if got, want := out.String(), "systemctl enable api\nsystemctl start api\n"; got != want {
		t.Fatalf("enable output = %q, want %q", got, want)
	}
}

func TestDisableCommandRuns(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		ActionStore: fakeStatusStore{
			services: []statuscmd.Service{
				{Name: "api", Enabled: true},
				{Name: "disabled", Enabled: false},
			},
		},
		ActionRunner: runner,
		Out:          &out,
	})
	cmd.SetArgs([]string{"disable", "--stop", "all"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected disable to run, got %v", err)
	}
	assertActionCall(t, runner, 0, "disable api")
	assertActionCall(t, runner, 1, "stop api")
	if got, want := out.String(), "systemctl disable api\nsystemctl stop api\n"; got != want {
		t.Fatalf("disable output = %q, want %q", got, want)
	}
}

func TestGroupListCommandRuns(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS:      "linux",
		GroupListStore: fakeGroupListStore{groups: map[string][]string{"web": {"api", "job"}}},
		Out:            &out,
	})
	cmd.SetArgs([]string{"group-list", "--services"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected group-list to run, got %v", err)
	}
	if got, want := out.String(), "web: api, job\n"; got != want {
		t.Fatalf("group-list output = %q, want %q", got, want)
	}
}

func TestGroupListAliasRuns(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS:      "linux",
		GroupListStore: fakeGroupListStore{groups: map[string][]string{"web": {"api"}, "worker": {"job"}}},
		Out:            &out,
	})
	cmd.SetArgs([]string{"list-group", "--one"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected list-group alias to run, got %v", err)
	}
	if got, want := out.String(), "web worker\n"; got != want {
		t.Fatalf("list-group output = %q, want %q", got, want)
	}
}

func TestGroupStartCommandRuns(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeActionRunner{}
	stopper := &fakeActionStopper{}
	cmd := New(Dependencies{
		RuntimeOS:      "linux",
		GroupListStore: fakeGroupListStore{groups: map[string][]string{"web": {"api", "job", "api"}}},
		ActionStore: fakeStatusStore{services: []statuscmd.Service{
			{Name: "job", Enabled: true},
			{Name: "api", Enabled: true},
		}},
		ActionRunner:      runner,
		ActionAutoStopper: stopper,
		Out:               &out,
	})
	cmd.SetArgs([]string{"start-group", "--stop-after", "30s", "web"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected group-start alias to run, got %v", err)
	}
	assertActionCall(t, runner, 0, "start api job")
	if got, want := out.String(), "systemctl start api job\n"; got != want {
		t.Fatalf("group-start output = %q, want %q", got, want)
	}
	if got, want := strings.Join(stopper.services, " "), "api job"; got != want {
		t.Fatalf("auto-stop services = %q, want %q", got, want)
	}
	if got, want := stopper.duration, 30*time.Second; got != want {
		t.Fatalf("auto-stop duration = %s, want %s", got, want)
	}
}

func TestGroupStartReportsMissingGroup(t *testing.T) {
	cmd := New(Dependencies{
		RuntimeOS:      "linux",
		GroupListStore: fakeGroupListStore{groups: map[string][]string{"web": {"api"}}},
	})
	cmd.SetArgs([]string{"group-start", "missing"})

	err := cmd.Execute()
	if err == nil || err.Error() != "cannot get services from group: group missing is not exist" {
		t.Fatalf("expected missing group error, got %v", err)
	}
}

func TestGroupStopAndRestartCommandRun(t *testing.T) {
	for _, tc := range []struct {
		args string
		want string
	}{
		{args: "stop-group", want: "stop api job"},
		{args: "group-restart", want: "restart api job"},
	} {
		runner := &fakeActionRunner{}
		cmd := New(Dependencies{
			RuntimeOS:      "linux",
			GroupListStore: fakeGroupListStore{groups: map[string][]string{"web": {"api", "job"}}},
			ActionStore: fakeStatusStore{services: []statuscmd.Service{
				{Name: "api", Enabled: true},
				{Name: "job", Enabled: true},
			}},
			ActionRunner: runner,
			Out:          io.Discard,
		})
		cmd.SetArgs([]string{tc.args, "web"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("%s failed: %v", tc.args, err)
		}
		assertActionCall(t, runner, 0, tc.want)
	}
}

func TestInstallServiceCommandRuns(t *testing.T) {
	file := writeCommandTestFile(t, "api.service", "[Unit]\nDescription=API\n[Service]\nExecStart=/bin/api\n")
	installer := &fakeInstallSvc{}
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS:           "linux",
		InstallSvcInstaller: installer,
		ActionRunner:        runner,
		Out:                 io.Discard,
	})
	cmd.SetArgs([]string{"install-service", "--no-start", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("install-service failed: %v", err)
	}
	if installer.name != "api" {
		t.Fatalf("installer name = %q, want api", installer.name)
	}
	assertActionCall(t, runner, 0, "daemon-reload")
	assertActionCall(t, runner, 1, "enable api")
}

func TestInstallCommandNoInstallRuns(t *testing.T) {
	var out bytes.Buffer
	cmd := New(Dependencies{RuntimeOS: "linux", Out: &out})
	cmd.SetArgs([]string{"install-command", "-n", "echo", "--no-install", "--", "/bin/echo", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("install-command failed: %v", err)
	}
	if !strings.Contains(out.String(), "# Service: echo.service") {
		t.Fatalf("install-command output = %q", out.String())
	}
}

func TestInstallDockerComposeAliasRuns(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "stack")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installer := &fakeInstallSvc{}
	cmd := New(Dependencies{
		RuntimeOS:           "linux",
		InstallSvcInstaller: installer,
		InstallDCLookPath:   func(file string) (string, error) { return "/usr/bin/docker", nil },
		Out:                 io.Discard,
	})
	cmd.SetArgs([]string{"idc", "--dir", filepath.Join(root, "opt"), "--no-enable", "--no-start", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("idc failed: %v", err)
	}
	if installer.name != "stack" || !strings.Contains(string(installer.data), "[X-Otter]") {
		t.Fatalf("installer = %q %q", installer.name, string(installer.data))
	}
}

func TestLinkServiceCommandRuns(t *testing.T) {
	root := t.TempDir()
	unit := filepath.Join(root, "systemd", "api.service")
	if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unit, []byte("[Service]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		FS: otterfs.New(otterfs.Config{
			ClassicServicePath: filepath.Join(root, "classic"),
			SystemdServicePath: filepath.Join(root, "systemd"),
		}),
		LinkStore: fakeStatusStore{services: []statuscmd.Service{{Name: "api", FragmentPath: unit}}},
	})
	cmd.SetArgs([]string{"fake", "api"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("link-service failed: %v", err)
	}
}

func TestReGenerateCommandRuns(t *testing.T) {
	runner := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS:           "linux",
		ReGenerateFinder:    fakePackageFinder{service: regeneratecmd.PackageService{Name: "api"}},
		ReGenerateGenerator: &fakePackageGenerator{},
		ActionRunner:        runner,
		Out:                 io.Discard,
	})
	cmd.SetArgs([]string{"regen", "--restart", "api"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	assertActionCall(t, runner, 0, "restart api")
}

func TestEditCommandRuns(t *testing.T) {
	runner := &fakeEditRunner{}
	system := &fakeActionRunner{}
	cmd := New(Dependencies{
		RuntimeOS: "linux",
		EditFinder: fakeServiceFinder{
			file: servicefile.File{Name: "api", Path: "/tmp/api.service", Source: servicefile.SourceClassic},
		},
		EditPrompter:     fakeEditPrompter{confirm: false},
		EditRunner:       runner,
		EditSystemRunner: system,
		EditLookPath:     func(file string) (string, error) { return "/usr/bin/" + file, nil },
		EditGetenv:       func(string) string { return "vim" },
		Out:              io.Discard,
	})
	cmd.SetArgs([]string{"edit", "api"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if runner.command != "/usr/bin/vim" {
		t.Fatalf("editor = %q", runner.command)
	}
	assertActionCall(t, system, 0, "daemon-reload")
}

func TestAuditCommandRuns(t *testing.T) {
	writer := &fakeAuditWriter{}
	cmd := New(Dependencies{
		RuntimeOS:   "linux",
		AuditWriter: writer,
		AuditEnviron: func() []string {
			return []string{"A=B"}
		},
	})
	cmd.SetArgs([]string{"audit", "--service-name", "api", "--action-name", "start"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("audit failed: %v", err)
	}
	if writer.record.Services[0] != "api" || writer.record.Actions[0] != "start" {
		t.Fatalf("record = %+v", writer.record)
	}
}

func TestAuditCommandMissingFlagsPrintsHint(t *testing.T) {
	writer := &fakeAuditWriter{}
	var out bytes.Buffer
	cmd := New(Dependencies{
		RuntimeOS:   "linux",
		AuditWriter: writer,
		Out:         &out,
	})
	cmd.SetArgs([]string{"audit"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("audit missing flags failed: %v", err)
	}
	if writer.record.Services != nil || writer.record.Actions != nil {
		t.Fatalf("record = %+v", writer.record)
	}
	if !strings.Contains(out.String(), "Maybe you want to use `otter-audit`?\n") ||
		strings.Contains(out.String(), "\x1b[31m") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestSelfCheckCommandRuns(t *testing.T) {
	runner := &fakeSelfCheckRunner{}
	cmd := New(Dependencies{
		RuntimeOS:           "linux",
		SelfCheckRunner:     runner,
		SelfCheckExecutable: func() (string, error) { return "/usr/bin/otter", nil },
		SelfCheckEnviron:    func() []string { return []string{"PATH=/bin", "AS=old"} },
	})
	cmd.SetArgs([]string{"self-check", "--deep", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected self-check to run, got %v", err)
	}
	if got, want := runner.file, "/usr/bin/otter"; got != want {
		t.Fatalf("self-check file = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.args, " "), "otter-self-check --deep api"; got != want {
		t.Fatalf("self-check args = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.env, " "), "PATH=/bin"; got != want {
		t.Fatalf("self-check env = %q, want %q", got, want)
	}
}

func TestSelfCheckAliasRuns(t *testing.T) {
	runner := &fakeSelfCheckRunner{}
	cmd := New(Dependencies{
		RuntimeOS:           "linux",
		SelfCheckRunner:     runner,
		SelfCheckExecutable: func() (string, error) { return "/usr/bin/otter", nil },
	})
	cmd.SetArgs([]string{"check", "--quick"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected check alias to run, got %v", err)
	}
	if got, want := strings.Join(runner.args, " "), "otter-self-check --quick"; got != want {
		t.Fatalf("check alias args = %q, want %q", got, want)
	}
}

func TestInstallHiddenCommandRuns(t *testing.T) {
	runner := &fakeSelfCheckRunner{}
	cmd := New(Dependencies{
		RuntimeOS:         "linux",
		InstallRunner:     runner,
		InstallExecutable: func() (string, error) { return "/usr/bin/otter", nil },
		InstallEnviron:    func() []string { return []string{"PATH=/bin", "AS=old"} },
	})
	cmd.SetArgs([]string{"install", "--force", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected install to run, got %v", err)
	}
	if got, want := runner.file, "/usr/bin/otter"; got != want {
		t.Fatalf("install file = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.args, " "), "otter-install --force api"; got != want {
		t.Fatalf("install args = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.env, " "), "PATH=/bin"; got != want {
		t.Fatalf("install env = %q, want %q", got, want)
	}
}

func TestUpsertSelfCommandRuns(t *testing.T) {
	runner := &fakeSelfCheckRunner{}
	mkdir := &fakeMkdirAll{}
	cmd := New(Dependencies{
		RuntimeOS:          "linux",
		UpsertSelfRunner:   runner,
		UpsertSelfPath:     func() string { return "/proc/self/exe" },
		UpsertSelfEnviron:  func() []string { return []string{"PATH=/bin", "AS=old"} },
		UpsertSelfMkdirAll: mkdir.mkdirAll,
	})
	cmd.SetArgs([]string{"upsert-self", "--force"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected upsert-self to run, got %v", err)
	}
	if got, want := mkdir.path, upsertselfcmd.LogBaseDir; got != want {
		t.Fatalf("upsert-self mkdir path = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.args, " "), "otter-upsert-self --force"; got != want {
		t.Fatalf("upsert-self args = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.env, " "), "PATH=/bin AS=otter-upsert-self"; got != want {
		t.Fatalf("upsert-self env = %q, want %q", got, want)
	}
}

func TestUpsertSelfAliasRuns(t *testing.T) {
	runner := &fakeSelfCheckRunner{}
	cmd := New(Dependencies{
		RuntimeOS:        "linux",
		UpsertSelfRunner: runner,
		UpsertSelfPath:   func() string { return "/proc/self/exe" },
	})
	cmd.SetArgs([]string{"us", "--quick"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected upsert-self alias to run, got %v", err)
	}
	if got, want := strings.Join(runner.args, " "), "otter-upsert-self --quick"; got != want {
		t.Fatalf("upsert-self alias args = %q, want %q", got, want)
	}
}

func TestUpsertClusterCommandRuns(t *testing.T) {
	runner := &fakeSelfCheckRunner{}
	mkdir := &fakeMkdirAll{}
	cmd := New(Dependencies{
		RuntimeOS:           "linux",
		UpsertClusterRunner: runner,
		UpsertClusterPath:   func() string { return "/proc/self/exe" },
		UpsertClusterEnv:    func() []string { return []string{"PATH=/bin", "AS=old"} },
		UpsertClusterMkdir:  mkdir.mkdirAll,
	})
	cmd.SetArgs([]string{"upsert-cluster", "--target", "api"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected upsert-cluster to run, got %v", err)
	}
	if got, want := mkdir.path, upsertclustercmd.LogBaseDir; got != want {
		t.Fatalf("upsert-cluster mkdir path = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.args, " "), "otter-upsert-cluster --target api"; got != want {
		t.Fatalf("upsert-cluster args = %q, want %q", got, want)
	}
	if got, want := strings.Join(runner.env, " "), "PATH=/bin AS=otter-upsert-cluster"; got != want {
		t.Fatalf("upsert-cluster env = %q, want %q", got, want)
	}
}

func TestUpsertClusterAliasRuns(t *testing.T) {
	runner := &fakeSelfCheckRunner{}
	cmd := New(Dependencies{
		RuntimeOS:           "linux",
		UpsertClusterRunner: runner,
		UpsertClusterPath:   func() string { return "/proc/self/exe" },
	})
	cmd.SetArgs([]string{"uc", "--quick"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected upsert-cluster alias to run, got %v", err)
	}
	if got, want := strings.Join(runner.args, " "), "otter-upsert-cluster --quick"; got != want {
		t.Fatalf("upsert-cluster alias args = %q, want %q", got, want)
	}
}

func assertCommand(t *testing.T, root *cobra.Command, name string) *cobra.Command {
	t.Helper()

	found, _, err := root.Find([]string{name})
	if err != nil {
		t.Fatalf("find command %q: %v", name, err)
	}
	if found == nil || found.Name() != name {
		t.Fatalf("find command %q resolved to %v", name, commandName(found))
	}
	return found
}

func assertPersistentFlag(t *testing.T, cmd *cobra.Command, name string) {
	t.Helper()
	if flag := cmd.PersistentFlags().Lookup(name); flag == nil {
		t.Fatalf("persistent flag %q not found", name)
	}
}

func assertFlags(t *testing.T, cmd *cobra.Command, names ...string) {
	t.Helper()
	for _, name := range names {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("command %q flag %q not found", cmd.Name(), name)
		}
	}
}

func assertActionCall(t *testing.T, runner *fakeActionRunner, index int, want string) {
	t.Helper()
	if index >= len(runner.calls) {
		t.Fatalf("action call %d missing, calls = %#v", index, runner.calls)
	}
	if got := strings.Join(runner.calls[index], " "); got != want {
		t.Fatalf("action call %d = %q, want %q", index, got, want)
	}
}

func commandName(cmd *cobra.Command) string {
	if cmd == nil {
		return "<nil>"
	}
	return cmd.Name()
}

type fakeStatusStore struct {
	services []statuscmd.Service
}

func (f fakeStatusStore) List(ctx context.Context) ([]statuscmd.Service, error) {
	return f.services, nil
}

type fakeDetailRunner struct {
	args []string
}

func (f *fakeDetailRunner) Run(
	ctx context.Context,
	args []string,
	in io.Reader,
	out io.Writer,
	errOut io.Writer,
) error {
	f.args = append([]string(nil), args...)
	return nil
}

var _ detailcmd.Runner = (*fakeDetailRunner)(nil)

type fakeActionRunner struct {
	calls [][]string
}

func (f *fakeActionRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	return nil
}

var _ startcmd.Runner = (*fakeActionRunner)(nil)

type fakeActionStopper struct {
	services []string
	duration time.Duration
}

func (f *fakeActionStopper) StopAfter(ctx context.Context, services []string, duration time.Duration) error {
	f.services = append([]string(nil), services...)
	f.duration = duration
	return nil
}

var _ startcmd.AutoStopper = (*fakeActionStopper)(nil)

type fakeInstallSvc struct {
	name string
	data []byte
}

func (f *fakeInstallSvc) Install(ctx context.Context, name string, data []byte) error {
	f.name = name
	f.data = append([]byte(nil), data...)
	return nil
}

var _ installservicecmd.Installer = (*fakeInstallSvc)(nil)

type fakeGroupListStore struct {
	groups map[string][]string
}

func (f fakeGroupListStore) List(ctx context.Context) (map[string][]string, error) {
	return f.groups, nil
}

var _ grouplistcmd.Store = fakeGroupListStore{}

type fakePackageFinder struct {
	service regeneratecmd.PackageService
}

func (f fakePackageFinder) FindPackage(ctx context.Context, serviceName string) (regeneratecmd.PackageService, error) {
	return f.service, nil
}

var _ regeneratecmd.Finder = fakePackageFinder{}

type fakePackageGenerator struct{}

func (f *fakePackageGenerator) Generate(ctx context.Context, service regeneratecmd.PackageService) error {
	return nil
}

var _ regeneratecmd.Generator = (*fakePackageGenerator)(nil)

type fakeEditRunner struct {
	command string
	args    []string
}

func (f *fakeEditRunner) Run(
	ctx context.Context,
	command string,
	args []string,
	in io.Reader,
	out io.Writer,
	errOut io.Writer,
) error {
	f.command = command
	f.args = append([]string(nil), args...)
	return nil
}

var _ editcmd.Runner = (*fakeEditRunner)(nil)

type fakeEditPrompter struct {
	confirm bool
}

func (f fakeEditPrompter) Choose(options []string) (string, error) {
	return "y", nil
}

func (f fakeEditPrompter) Confirm(prompt string) (bool, error) {
	return f.confirm, nil
}

func (f fakeEditPrompter) Input(prompt string) (string, error) {
	return "message", nil
}

var _ editcmd.Prompter = fakeEditPrompter{}

type fakeAuditWriter struct {
	record auditcmd.Record
}

func (f *fakeAuditWriter) Write(ctx context.Context, record auditcmd.Record) error {
	f.record = record
	return nil
}

var _ auditcmd.Writer = (*fakeAuditWriter)(nil)

type fakeShowPropertyGetter struct {
	serviceName string
	properties  showpropertycmd.Properties
}

func (f *fakeShowPropertyGetter) Get(ctx context.Context, serviceName string) (showpropertycmd.Properties, error) {
	f.serviceName = serviceName
	return f.properties, nil
}

func writeCommandTestFile(t *testing.T, name string, data string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return p
}

type fakeServiceFinder struct {
	file servicefile.File
	err  error
}

func (f fakeServiceFinder) Find(ctx context.Context, serviceName string) (servicefile.File, error) {
	return f.file, f.err
}

type fakeLogRunner struct {
	command string
	args    []string
}

func (f *fakeLogRunner) Run(ctx context.Context, command string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.command = command
	f.args = append([]string(nil), args...)
	return nil
}

func commandTestLookPath(file string) (string, error) {
	if file == "journalctl" {
		return "/bin/journalctl", nil
	}
	return "", errors.New("missing")
}

var _ logcmd.Runner = (*fakeLogRunner)(nil)

type fakeDaemonReloadRunner struct {
	args []string
}

func (f *fakeDaemonReloadRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.args = append([]string(nil), args...)
	return nil
}

var _ daemonreloadcmd.Runner = (*fakeDaemonReloadRunner)(nil)

type fakeShowPidsFinder struct {
	pids map[string][]int32
}

func (f fakeShowPidsFinder) Find(ctx context.Context, serviceName string) ([]int32, error) {
	return f.pids[serviceName], nil
}

var _ showpidscmd.Finder = fakeShowPidsFinder{}

type fakeShowPortsConnFinder struct {
	connections map[int32][]showportscmd.Connection
}

func (f fakeShowPortsConnFinder) Find(ctx context.Context, pid int32) ([]showportscmd.Connection, error) {
	return f.connections[pid], nil
}

var _ showportscmd.ConnFinder = fakeShowPortsConnFinder{}

type fakeSelfCheckRunner struct {
	file string
	args []string
	env  []string
}

func (f *fakeSelfCheckRunner) Exec(ctx context.Context, file string, args []string, env []string) error {
	f.file = file
	f.args = append([]string(nil), args...)
	f.env = append([]string(nil), env...)
	return nil
}

var _ selfcheckcmd.Runner = (*fakeSelfCheckRunner)(nil)
var _ installcmd.Runner = (*fakeSelfCheckRunner)(nil)
var _ upsertselfcmd.Runner = (*fakeSelfCheckRunner)(nil)
var _ upsertclustercmd.Runner = (*fakeSelfCheckRunner)(nil)

type fakeMkdirAll struct {
	path string
	perm os.FileMode
}

func (f *fakeMkdirAll) mkdirAll(path string, perm os.FileMode) error {
	f.path = path
	f.perm = perm
	return nil
}
