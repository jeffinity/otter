package main

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffinity/otter/pkg/logx"
)

var rootCmd = &cobra.Command{
	Use: "newapp",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			logx.ErrorErr(err, "展示帮助信息失败")
		}
	},
}

func init() {
	rootCmd.AddCommand(CmdVersion())
	rootCmd.AddCommand(CmdNew())
}

func main() {
	logx.Init(logx.Config{
		Prefix:     "newapp",
		Timestamp:  true,
		Caller:     false,
		TimeFormat: time.Kitchen,
	})

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logx.SetPrefix(commandPrefix(cmd))
		if cmd.Name() != "version" {
			ShowInfo()
		}
	}

	err := rootCmd.Execute()
	if err != nil {
		logx.ErrorErr(err, "命令执行失败")
	}
}

func commandPrefix(cmd *cobra.Command) string {
	switch cmd.Name() {
	case "new":
		return "🆕 new"
	case "version":
		return "version"
	default:
		return cmd.Name()
	}
}
