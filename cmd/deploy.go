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
		Use: "deploy",
	}
	flags := cmd.Flags()

	app := cli.NewApp()

	flags.StringVar(&app.SourceS3Bucket, "source-bucket", "func-source", "S3 Bucket to use for source code")

	stack := flags.StringP("stack", "s", "", "CloudFormation stack name")
	cmd.MarkFlagRequired("stack")

	logLevel := flags.CountP("v", "v", "Log level")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		app.Logger = cli.NewLogger(*logLevel)

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		ctx := context.Background()
		code := app.Deploy(ctx, dir, *stack)
		os.Exit(code)
	}

	return cmd
}
