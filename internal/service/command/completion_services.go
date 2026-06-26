package command

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func (b *commandBuilder) completeServices(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_ = cmd
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := map[string]struct{}{}
	addServiceNames(names, filepath.Join(b.deps.FS.ClassicServicePath(), "*.service"), toComplete)
	addServiceNames(names, filepath.Join(b.deps.FS.PackageServicePath(), "*", "*", "*.service"), toComplete)

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, cobra.ShellCompDirectiveNoFileComp
}

func addServiceNames(names map[string]struct{}, pattern string, prefix string) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, match := range matches {
		base := filepath.Base(match)
		if !strings.HasSuffix(base, ".service") || strings.HasPrefix(base, ".") {
			continue
		}
		name := strings.TrimSuffix(base, ".service")
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		names[name] = struct{}{}
	}
}
