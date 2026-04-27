package servicecmd

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
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
	cmd := New(Dependencies{RuntimeOS: "linux"})

	err := cmd.Execute()
	if err == nil || err.Error() != "otter service status is not implemented yet" {
		t.Fatalf("expected default status dispatch, got %v", err)
	}
}

func TestServiceCommandTree(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})

	assertPersistentFlag(t, cmd, "cluster")

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
}

func TestStatusRejectsConflictingSortFlags(t *testing.T) {
	cmd := New(Dependencies{RuntimeOS: "linux"})
	cmd.SetArgs([]string{"status", "--asc", "--desc"})

	err := cmd.Execute()
	if err == nil || err.Error() != "--asc and --desc cannot be apply in the meantime" {
		t.Fatalf("expected conflicting sort flags error, got %v", err)
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

func commandName(cmd *cobra.Command) string {
	if cmd == nil {
		return "<nil>"
	}
	return cmd.Name()
}
