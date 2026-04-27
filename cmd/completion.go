package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var completionShells = []string{"bash", "zsh", "fish", "powershell"}

var configCompletionShells = []string{"bash", "zsh", "fish"}

type configCompletionOptions struct {
	installDir string
	system     bool
}

type configCompletionDeps struct {
	RuntimeOS string
	HomeDir   string
	ShellPath string
}

func CmdCompletion() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "completion [bash|zsh|fish|powershell]",
		Short:             "Generate shell completion script",
		Args:              validateShellArg(completionShells, true),
		ValidArgsFunction: completeShells(completionShells),
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeCompletionScript(cmd.Root(), args[0], cmd.OutOrStdout())
		},
	}
	return cmd
}

func CmdConfigCompletion() *cobra.Command {
	return newConfigCompletionCommand(configCompletionDeps{
		RuntimeOS: runtime.GOOS,
		ShellPath: os.Getenv("SHELL"),
	})
}

func newConfigCompletionCommand(deps configCompletionDeps) *cobra.Command {
	opts := &configCompletionOptions{}
	cmd := &cobra.Command{
		Use:               "config-completion [bash|zsh|fish]",
		Short:             "Install shell completion for Linux shells",
		Args:              validateShellArg(configCompletionShells, false),
		ValidArgsFunction: completeShells(configCompletionShells),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.RuntimeOS == "" {
				deps.RuntimeOS = runtime.GOOS
			}
			if deps.RuntimeOS != "linux" {
				return fmt.Errorf("otter config-completion is only supported on linux")
			}

			shell := ""
			if len(args) > 0 {
				shell = args[0]
			} else {
				shell = detectShell(deps.ShellPath)
			}
			if !slices.Contains(configCompletionShells, shell) {
				return fmt.Errorf("unsupported shell %q, supported shells: %s", shell, strings.Join(configCompletionShells, ", "))
			}

			homeDir := deps.HomeDir
			if homeDir == "" {
				var err error
				homeDir, err = os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("detect home directory failed: %w", err)
				}
			}

			installDir := opts.installDir
			if installDir == "" {
				installDir = defaultCompletionDir(shell, opts.system, homeDir)
			}

			var script bytes.Buffer
			if err := writeCompletionScript(cmd.Root(), shell, &script); err != nil {
				return err
			}

			if err := os.MkdirAll(installDir, 0o755); err != nil {
				return fmt.Errorf("create completion directory failed: %w", err)
			}
			installPath := filepath.Join(installDir, completionFileName(shell))
			if err := os.WriteFile(installPath, script.Bytes(), 0o644); err != nil {
				return fmt.Errorf("write completion file failed: %w", err)
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "installed %s completion: %s\n%s\n", shell, installPath, completionHint(shell, installDir, opts.system))
			return err
		},
	}
	cmd.Flags().StringVar(&opts.installDir, "dir", "", "completion install directory")
	cmd.Flags().BoolVar(&opts.system, "system", false, "install to a common system-wide completion directory")
	_ = cmd.MarkFlagDirname("dir")
	return cmd
}

func validateShellArg(shells []string, required bool) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if required && len(args) != 1 {
			return fmt.Errorf("requires one shell: %s", strings.Join(shells, ", "))
		}
		if !required && len(args) > 1 {
			return fmt.Errorf("accepts at most one shell: %s", strings.Join(shells, ", "))
		}
		if len(args) == 0 {
			return nil
		}
		if !slices.Contains(shells, args[0]) {
			return fmt.Errorf("unsupported shell %q, supported shells: %s", args[0], strings.Join(shells, ", "))
		}
		return nil
	}
}

func completeShells(shells []string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		matches := make([]string, 0, len(shells))
		for _, shell := range shells {
			if strings.HasPrefix(shell, toComplete) {
				matches = append(matches, shell)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

func writeCompletionScript(root *cobra.Command, shell string, out io.Writer) error {
	switch shell {
	case "bash":
		return root.GenBashCompletion(out)
	case "zsh":
		return root.GenZshCompletion(out)
	case "fish":
		return root.GenFishCompletion(out, true)
	case "powershell":
		return root.GenPowerShellCompletion(out)
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}

func detectShell(shellPath string) string {
	name := filepath.Base(shellPath)
	switch name {
	case "bash", "zsh", "fish":
		return name
	default:
		return ""
	}
}

func defaultCompletionDir(shell string, system bool, homeDir string) string {
	if system {
		switch shell {
		case "bash":
			return "/etc/bash_completion.d"
		case "zsh":
			return "/usr/local/share/zsh/site-functions"
		case "fish":
			return "/etc/fish/completions"
		}
	}

	switch shell {
	case "bash":
		return filepath.Join(homeDir, ".local", "share", "bash-completion", "completions")
	case "zsh":
		return filepath.Join(homeDir, ".local", "share", "zsh", "site-functions")
	case "fish":
		return filepath.Join(homeDir, ".config", "fish", "completions")
	default:
		return homeDir
	}
}

func completionFileName(shell string) string {
	switch shell {
	case "zsh":
		return "_otter"
	case "fish":
		return "otter.fish"
	default:
		return "otter"
	}
}

func completionHint(shell, installDir string, system bool) string {
	if system {
		return "restart your shell to load the completion script"
	}

	switch shell {
	case "bash":
		return "restart bash, or run in current shell: source " + filepath.Join(installDir, "otter") + "\ndo not execute the completion file with bash or as a program"
	case "zsh":
		return "ensure this directory is in fpath before compinit: " + installDir
	case "fish":
		return "restart fish, or run: source " + filepath.Join(installDir, "otter.fish")
	default:
		return "restart your shell to load the completion script"
	}
}
