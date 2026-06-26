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
	installDir   string
	system       bool
	serviceAlias string
	bashRCPath   string
	zshRCPath    string
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
			return runConfigCompletion(cmd, args, deps, opts)
		},
	}
	cmd.Flags().StringVar(&opts.installDir, "dir", "", "completion install directory")
	cmd.Flags().BoolVar(&opts.system, "system", false, "install to a common system-wide completion directory")
	cmd.Flags().StringVar(&opts.serviceAlias, "service-alias", "", "configure shell alias for 'otter service' with completion (e.g. os)")
	cmd.Flags().StringVar(&opts.bashRCPath, "bashrc", "", "bash rc file path used with --service-alias (default: ~/.bashrc)")
	cmd.Flags().StringVar(&opts.zshRCPath, "zshrc", "", "zsh rc file path used with --service-alias (default: ~/.zshrc)")
	_ = cmd.MarkFlagDirname("dir")
	_ = cmd.MarkFlagFilename("bashrc")
	_ = cmd.MarkFlagFilename("zshrc")
	return cmd
}

func runConfigCompletion(cmd *cobra.Command, args []string, deps configCompletionDeps, opts *configCompletionOptions) error {
	if deps.RuntimeOS == "" {
		deps.RuntimeOS = runtime.GOOS
	}
	if deps.RuntimeOS != "linux" {
		return fmt.Errorf("otter config-completion is only supported on linux")
	}
	shell := resolveConfigShell(args, deps.ShellPath)
	if err := validateConfigCompletionOptions(shell, opts); err != nil {
		return err
	}
	homeDir, err := resolveHomeDir(deps.HomeDir)
	if err != nil {
		return err
	}
	installDir := resolveInstallDir(opts.installDir, shell, opts.system, homeDir)
	if err := installCompletionScript(cmd.Root(), shell, installDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "installed %s completion: %s\n%s\n", shell, filepath.Join(installDir, completionFileName(shell)), completionHint(shell, installDir, opts.system)); err != nil {
		return err
	}
	return configureServiceAlias(cmd, shell, homeDir, opts)
}

func resolveConfigShell(args []string, shellPath string) string {
	if len(args) > 0 {
		return args[0]
	}
	return detectShell(shellPath)
}

func validateConfigCompletionOptions(shell string, opts *configCompletionOptions) error {
	if !slices.Contains(configCompletionShells, shell) {
		return fmt.Errorf("unsupported shell %q, supported shells: %s", shell, strings.Join(configCompletionShells, ", "))
	}
	if opts.serviceAlias != "" && !validBashAliasName(opts.serviceAlias) {
		return fmt.Errorf("invalid --service-alias %q, only letters, digits and underscore are allowed", opts.serviceAlias)
	}
	if opts.bashRCPath != "" && shell != "bash" {
		return fmt.Errorf("--bashrc can only be used with bash")
	}
	if opts.zshRCPath != "" && shell != "zsh" {
		return fmt.Errorf("--zshrc can only be used with zsh")
	}
	return nil
}

func resolveHomeDir(homeDir string) (string, error) {
	if homeDir != "" {
		return homeDir, nil
	}
	detected, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect home directory failed: %w", err)
	}
	return detected, nil
}

func resolveInstallDir(installDir string, shell string, system bool, homeDir string) string {
	if installDir != "" {
		return installDir
	}
	return defaultCompletionDir(shell, system, homeDir)
}

func installCompletionScript(root *cobra.Command, shell string, installDir string) error {
	var script bytes.Buffer
	if err := writeCompletionScript(root, shell, &script); err != nil {
		return err
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("create completion directory failed: %w", err)
	}
	installPath := filepath.Join(installDir, completionFileName(shell))
	if err := os.WriteFile(installPath, script.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write completion file failed: %w", err)
	}
	return nil
}

func configureServiceAlias(cmd *cobra.Command, shell string, homeDir string, opts *configCompletionOptions) error {
	if opts.serviceAlias == "" {
		return nil
	}
	switch shell {
	case "bash":
		bashRCPath := opts.bashRCPath
		if bashRCPath == "" {
			bashRCPath = filepath.Join(homeDir, ".bashrc")
		}
		block := buildBashServiceAliasBlock(opts.serviceAlias)
		if err := upsertManagedBlock(bashRCPath, "otter-service-alias-completion", block); err != nil {
			return fmt.Errorf("configure bash alias completion failed: %w", err)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "configured bash alias completion in %s: %s\n", bashRCPath, opts.serviceAlias)
		return err
	case "zsh":
		zshRCPath := opts.zshRCPath
		if zshRCPath == "" {
			zshRCPath = filepath.Join(homeDir, ".zshrc")
		}
		block := buildZshServiceAliasBlock(opts.serviceAlias)
		if err := upsertManagedBlock(zshRCPath, "otter-service-alias-completion", block); err != nil {
			return fmt.Errorf("configure zsh alias completion failed: %w", err)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "configured zsh alias completion in %s: %s\n", zshRCPath, opts.serviceAlias)
		return err
	default:
		return fmt.Errorf("--service-alias is only supported for bash and zsh")
	}
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

func buildBashServiceAliasBlock(aliasName string) string {
	return strings.TrimSpace(fmt.Sprintf(`
alias %s='otter service'
_otter_service_alias_complete_%s() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local args=("${COMP_WORDS[@]:1}")
  local out line

  out="$(otter __complete service "${args[@]}" 2>/dev/null)" || return 0
  COMPREPLY=()
  while IFS= read -r line; do
    [[ "$line" == :* ]] && continue
    line="${line%%$'\t'*}"
    [[ -z "$line" ]] && continue
    COMPREPLY+=("$line")
  done <<< "$out"

  if [[ -n "$cur" ]]; then
    local filtered=() c
    for c in "${COMPREPLY[@]}"; do
      [[ "$c" == "$cur"* ]] && filtered+=("$c")
    done
    COMPREPLY=("${filtered[@]}")
  fi
}
complete -o default -F _otter_service_alias_complete_%s %s
`, aliasName, aliasName, aliasName, aliasName))
}

func buildZshServiceAliasBlock(aliasName string) string {
	return strings.TrimSpace(fmt.Sprintf(`
alias %s='otter service'
_otter_service_alias_complete_%s() {
  local idx
  local -a words_copy
  words_copy=("${words[@]}")
  words=(otter service)
  for (( idx = 2; idx <= ${#words_copy[@]}; idx++ )); do
    words+=("${words_copy[idx]}")
  done
  (( CURRENT += 1 ))
  _otter
}
compdef _otter_service_alias_complete_%s %s
`, aliasName, aliasName, aliasName, aliasName))
}

func upsertManagedBlock(path string, tag string, block string) error {
	start := "# >>> otter:" + tag + " >>>"
	end := "# <<< otter:" + tag + " <<<"
	payload := start + "\n" + strings.TrimSpace(block) + "\n" + end + "\n"

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(data)

	startIdx := strings.Index(content, start)
	endIdx := strings.Index(content, end)
	switch {
	case startIdx >= 0 && endIdx > startIdx:
		endIdx += len(end)
		replaced := content[:startIdx] + payload + strings.TrimLeft(content[endIdx:], "\n")
		return os.WriteFile(path, []byte(strings.TrimRight(replaced, "\n")+"\n"), 0o644)
	case startIdx >= 0 || endIdx >= 0:
		// broken block, rewrite by appending one clean block
		content = strings.TrimRight(content, "\n")
		if content != "" {
			content += "\n\n"
		}
		content += payload
		return os.WriteFile(path, []byte(content), 0o644)
	default:
		content = strings.TrimRight(content, "\n")
		if content != "" {
			content += "\n\n"
		}
		content += payload
		return os.WriteFile(path, []byte(content), 0o644)
	}
}

func validBashAliasName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if r == '_' || (r >= '0' && r <= '9' && i > 0) || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		return false
	}
	return true
}
