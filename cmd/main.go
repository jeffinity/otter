package main

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffinity/otter/pkg/logx"
)

var rootCmd = &cobra.Command{
	Use: "otter",
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
	rootCmd.AddCommand(CmdService())
	rootCmd.AddCommand(CmdCompletion())
	rootCmd.AddCommand(CmdConfigCompletion())
}

func main() {
	logx.Init(logx.Config{
		Prefix:     "otter",
		Timestamp:  true,
		Caller:     false,
		TimeFormat: time.Kitchen,
	})

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logx.SetPrefix(commandPrefix(cmd))
		if cmd.Name() == "new" {
			ShowInfo()
		}
	}

	err := rootCmd.Execute()
	if err != nil {
		logx.ErrorErr(err, "命令执行失败")
		os.Exit(1)
	}
}

func commandPrefix(cmd *cobra.Command) string {
	switch cmd.Name() {
	case "new":
		return "🆕 new"
	case "version":
		return "version"
	case "service":
		return "service"
	default:
		return cmd.Name()
	}
}
