package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	appnew "github.com/jeffinity/otter/internal/newapp"
	"github.com/jeffinity/otter/pkg/logx"
	"github.com/jeffinity/otter/pkg/tuix"
)

type newOptions struct {
	layoutSource string
	outputDir    string
	monoRepo     bool
}

type parsedNewArgs struct {
	modulePath string
	appName    string
}

func CmdNew() *cobra.Command {
	opts := &newOptions{}
	cmd := &cobra.Command{
		Use:   "new [选项] <模块路径> <应用名>",
		Short: "基于 app-layout 模板创建新项目",
		Long: strings.Join([]string{
			"创建应用脚手架，支持单仓和大仓两种模式。",
			"单仓模式（默认）：参数必须为 <模块路径> <应用名>。",
			"大仓模式（-m）：支持两种入参形式：",
			"  1) 指定包名：<模块路径> <应用名>",
			"  2) 仅应用名：<应用名>（自动读取当前模块路径）",
		}, "\n"),
		Example: strings.Join([]string{
			"  # 单仓模式（默认）",
			"  newapp new github.com/acme/order order-api",
			"  newapp new -o /tmp/workspace github.com/acme/order order-api",
			"",
			"  # 大仓模式（-m，方式一：指定包名，创建独立仓）",
			"  newapp new -m -o /path/to/output github.com/acme/mono order-api",
			"",
			"  # 大仓模式（-m，方式二：仅应用名，写入当前 app/）",
			"  newapp new -m order-api",
			"",
			"  # 指定 layout 源",
			"  newapp new -r /Users/jeff/tao/workspace/app-layout github.com/acme/order order-api",
		}, "\n"),
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := parseAndValidateArgs(opts.monoRepo, args)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parseAndValidateArgs(opts.monoRepo, args)
			if err != nil {
				return err
			}

			return appnew.Run(parsed.modulePath, parsed.appName, appnew.Options{
				LayoutSource: opts.layoutSource,
				OutputDir:    opts.outputDir,
				MonoRepo:     opts.monoRepo,
			})
		},
	}

	cmd.SetUsageFunc(func(c *cobra.Command) error {
		return renderNewUsage(c)
	})
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		if err := renderNewUsage(c); err != nil {
			logx.ErrorErr(err, "渲染帮助信息失败")
		}
	})
	cmd.Flags().StringVarP(&opts.layoutSource, "repo", "r", appnew.DefaultLayoutRepo, "layout 源（支持 git 地址或本地目录）")
	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", ".", "输出基目录（单仓模式会在其下创建 <应用名> 目录）")
	cmd.Flags().BoolVarP(&opts.monoRepo, "mono", "m", false, "启用大仓模式（输出目录中必须已存在 app 子目录）")
	return cmd
}

func renderNewUsage(cmd *cobra.Command) error {
	usageLines := []string{
		"单仓模式（默认）:",
		"  newapp new [选项] <模块路径> <应用名>",
		"",
		"大仓模式（-m，创建独立仓）:",
		"  newapp new -m [选项] <模块路径> <应用名>",
		"",
		"大仓模式（-m，向现有 app/ 新增应用）:",
		"  newapp new -m [选项] <应用名>",
	}

	exampleLines := []string{
		"newapp new github.com/acme/order order-api",
		"newapp new -o /tmp/work github.com/acme/order order-api",
		"newapp new -m -o /path/to/output github.com/acme/mono order-api",
		"newapp new -m order-api",
		"newapp new -r /Users/jeff/tao/workspace/app-layout github.com/acme/order order-api",
	}

	flagUsage := strings.TrimSpace(cmd.LocalFlags().FlagUsages())
	flagLines := splitLines(flagUsage)
	if len(flagLines) == 0 {
		flagLines = []string{"无"}
	}

	content := tuix.RenderUsage("newapp new", "创建应用脚手架（支持单仓与大仓）", []tuix.Section{
		{Title: "用法", Lines: usageLines},
		{Title: "示例", Lines: exampleLines},
		{Title: "选项", Lines: flagLines},
	})
	return tuix.PrintStatic(cmd.OutOrStdout(), content)
}

func hasSpace(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	lines := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		lines = append(lines, p)
	}
	return lines
}

func parseAndValidateArgs(monoRepo bool, args []string) (parsedNewArgs, error) {
	parsed, err := parseNewArgs(monoRepo, args)
	if err != nil {
		return parsedNewArgs{}, err
	}

	if parsed.modulePath != "" && strings.TrimSpace(parsed.modulePath) == "" {
		return parsedNewArgs{}, fmt.Errorf("参数错误：<模块路径> 不能为空")
	}
	if strings.TrimSpace(parsed.appName) == "" {
		return parsedNewArgs{}, fmt.Errorf("参数错误：<应用名> 不能为空")
	}
	if hasSpace(parsed.appName) {
		return parsedNewArgs{}, fmt.Errorf("参数错误：应用名不能包含空白字符")
	}
	if strings.Contains(parsed.appName, "/") || strings.Contains(parsed.appName, "\\") {
		return parsedNewArgs{}, fmt.Errorf("参数错误：应用名不能包含路径分隔符")
	}
	return parsed, nil
}

func parseNewArgs(monoRepo bool, args []string) (parsedNewArgs, error) {
	if monoRepo {
		switch len(args) {
		case 1:
			return parsedNewArgs{appName: args[0]}, nil
		case 2:
			return parsedNewArgs{modulePath: args[0], appName: args[1]}, nil
		default:
			return parsedNewArgs{}, fmt.Errorf("参数错误：大仓模式需要 [模块路径] <应用名>")
		}
	}

	if len(args) != 2 {
		return parsedNewArgs{}, fmt.Errorf("参数错误：单仓模式需要 <模块路径> 和 <应用名>")
	}
	return parsedNewArgs{modulePath: args[0], appName: args[1]}, nil
}
