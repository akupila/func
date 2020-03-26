package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/func/func/version"
	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "func\n")
			fmt.Fprintf(os.Stdout, "  Version:    %s\n", version.Version)
			fmt.Fprintf(os.Stdout, "  Build date: %s\n", version.BuildDate)
			fmt.Fprintf(os.Stdout, "  Go version: %s\n", runtime.Version())
		},
	}
	return cmd
}
