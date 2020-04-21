package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/func/func/cli"
	"github.com/spf13/cobra"
)

func deployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy CloudFormation stack",
	}
	flags := cmd.Flags()
	verbose := flags.Bool("verbose", false, "Enable verbose output")

	var opts cli.DeploymentOpts
	flags.StringVarP(&opts.StackName, "stack", "s", "", "CloudFormation stack name")
	flags.StringVar(&opts.SourceBucket, "source-bucket", "", "S3 Bucket to use for source code")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		app := cli.NewApp(*verbose)

		ctx := context.Background()
		code := app.DeployCloudFormation(ctx, dir, opts)
		os.Exit(code)
	}

	return cmd
}
