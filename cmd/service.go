package main

import (
	"github.com/spf13/cobra"

	"github.com/jeffinity/otter/internal/servicecmd"
)

func CmdService() *cobra.Command {
	return servicecmd.New(servicecmd.Dependencies{})
}
