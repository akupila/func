package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Exec executes the main command.
func Exec() {
	cmd := &cobra.Command{
		Use: "func",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, cmd.UsageString())
		},
	}

	cmd.AddCommand(versionCommand())
	cmd.AddCommand(generateCommand())
	cmd.AddCommand(deployCommand())

	_ = cmd.Execute()
}
