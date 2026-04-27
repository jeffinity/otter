package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCommandGeneratesBash(t *testing.T) {
	root := testCompletionRoot(t.TempDir(), "linux", "/bin/bash")

	out, err := executeTestCommand(root, "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash returned error: %v", err)
	}
	if !strings.Contains(out, "__start_otter") {
		t.Fatalf("bash completion should contain otter entrypoint, got: %s", out)
	}
	if !strings.Contains(out, "config-completion") {
		t.Fatalf("bash completion should include config-completion command")
	}
}

func TestCompletionCommandTree(t *testing.T) {
	root := testCompletionRoot(t.TempDir(), "linux", "/bin/bash")

	completion := assertTestCommand(t, root, "completion")
	if completion.Hidden {
		t.Fatalf("completion should not be hidden")
	}

	configCompletion := assertTestCommand(t, root, "config-completion")
	if configCompletion.Hidden {
		t.Fatalf("config-completion should not be hidden")
	}
	for _, name := range []string{"dir", "system"} {
		if flag := configCompletion.Flags().Lookup(name); flag == nil {
			t.Fatalf("config-completion flag %q not found", name)
		}
	}
}

func TestConfigCompletionInstallsUserBash(t *testing.T) {
	homeDir := t.TempDir()
	root := testCompletionRoot(homeDir, "linux", "/bin/bash")

	out, err := executeTestCommand(root, "config-completion", "bash")
	if err != nil {
		t.Fatalf("config-completion bash returned error: %v", err)
	}

	installPath := filepath.Join(homeDir, ".local", "share", "bash-completion", "completions", "otter")
	assertFileContains(t, installPath, "__start_otter")
	if !strings.Contains(out, installPath) {
		t.Fatalf("output should include install path, got: %s", out)
	}
	if !strings.Contains(out, "run in current shell: source "+installPath) {
		t.Fatalf("output should explain how to load bash completion, got: %s", out)
	}
	if !strings.Contains(out, "do not execute the completion file with bash or as a program") {
		t.Fatalf("output should warn against executing bash completion file, got: %s", out)
	}
}

func TestConfigCompletionDetectsShellAndCustomDir(t *testing.T) {
	homeDir := t.TempDir()
	installDir := filepath.Join(t.TempDir(), "completions")
	root := testCompletionRoot(homeDir, "linux", "/usr/bin/fish")

	_, err := executeTestCommand(root, "config-completion", "--dir", installDir)
	if err != nil {
		t.Fatalf("config-completion detected fish returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(installDir, "otter.fish"), "complete -c otter")
}

func TestConfigCompletionRejectsUnsupportedPlatform(t *testing.T) {
	root := testCompletionRoot(t.TempDir(), "darwin", "/bin/bash")

	_, err := executeTestCommand(root, "config-completion", "bash")
	if err == nil || err.Error() != "otter config-completion is only supported on linux" {
		t.Fatalf("expected unsupported platform error, got: %v", err)
	}
}

func testCompletionRoot(homeDir, runtimeOS, shellPath string) *cobra.Command {
	root := &cobra.Command{
		Use:          "otter",
		SilenceUsage: true,
	}
	root.AddCommand(CmdCompletion())
	root.AddCommand(newConfigCompletionCommand(configCompletionDeps{
		RuntimeOS: runtimeOS,
		HomeDir:   homeDir,
		ShellPath: shellPath,
	}))
	return root
}

func executeTestCommand(cmd *cobra.Command, args ...string) (string, error) {
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func assertFileContains(t *testing.T, path string, needle string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s failed: %v", path, err)
	}
	if !strings.Contains(string(data), needle) {
		t.Fatalf("file %s should contain %q, got: %s", path, needle, string(data))
	}
}

func assertTestCommand(t *testing.T, root *cobra.Command, name string) *cobra.Command {
	t.Helper()
	found, _, err := root.Find([]string{name})
	if err != nil {
		t.Fatalf("find command %q failed: %v", name, err)
	}
	if found == nil || found.Name() != name {
		t.Fatalf("find command %q resolved to %v", name, found)
	}
	return found
}
