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

	logLevel := flags.CountP("v", "v", "Log level")

	var opts cli.DeploymentOpts
	flags.StringVarP(&opts.StackName, "stack", "s", "", "CloudFormation stack name")
	flags.StringVar(&opts.SourceBucket, "source-bucket", "", "S3 Bucket to use for source code")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		app := cli.NewApp(cli.LogLevel(*logLevel))

		ctx := context.Background()
		code := app.DeployCloudFormation(ctx, dir, opts)
		os.Exit(code)
	}

	return cmd
}
