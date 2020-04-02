package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/func/func/cli"
	"github.com/spf13/cobra"
)

func generateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate CloudFormation template",
	}
	flags := cmd.Flags()

	logLevel := flags.CountP("v", "v", "Log level")

	var opts cli.GenerateCloudFormationOpts
	flags.StringVarP(&opts.Format, "format", "f", "yaml", "Output format")
	flags.StringVar(&opts.SourceBucket, "source-bucket", "", "S3 Bucket to use for source code")
	flags.BoolVar(&opts.ProcessSource, "process-source", false, "Build and upload source code if needed")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		app := cli.NewApp(cli.LogLevel(*logLevel))

		ctx := context.Background()
		code := app.GenerateCloudFormation(ctx, dir, opts)
		os.Exit(code)
	}

	return cmd
}
