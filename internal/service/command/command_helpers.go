package command

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func output(cmd *cobra.Command, out io.Writer) io.Writer {
	if out != nil {
		return out
	}
	return cmd.OutOrStdout()
}

func errOutput(cmd *cobra.Command, out io.Writer) io.Writer {
	if out != nil {
		return out
	}
	return cmd.ErrOrStderr()
}

func input(cmd *cobra.Command, in io.Reader) io.Reader {
	if in != nil {
		return in
	}
	return cmd.InOrStdin()
}

func errServiceNameRequired() error {
	return fmt.Errorf("service name is required")
}
