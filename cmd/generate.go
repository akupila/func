package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/func/func/cli"
	"github.com/spf13/cobra"
)

func generateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "generate",
	}
	flags := cmd.Flags()

	app := cli.NewApp()
	flags.StringVar(&app.SourceS3Bucket, "source-bucket", "func-source", "S3 Bucket to use for source code")

	logLevel := flags.CountP("v", "v", "Log level")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		app.Logger = cli.NewLogger(*logLevel)

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		ctx := context.Background()

		tmpl, code := app.GenerateCloudFormation(ctx, dir)
		if code != 0 {
			os.Exit(code)
		}

		j, err := json.MarshalIndent(tmpl, "", "    ")
		if err != nil {
			panic(err)
		}

		fmt.Fprintln(os.Stdout, string(j))
	}

	return cmd
}
