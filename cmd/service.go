package main

import (
	"github.com/spf13/cobra"

	servicecmd "github.com/jeffinity/otter/internal/service/command"
)

func CmdService() *cobra.Command {
	return servicecmd.New(servicecmd.Dependencies{})
}
