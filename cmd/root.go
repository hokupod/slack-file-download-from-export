/*
Copyright © 2022 Hokuto Takemiya <hokupod@outlook.com>

*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hokupod/slack-file-download-from-export/sfd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "slack-file-download-from-export",
	Short: "A tool to batch download files that were attached to slack.",
	Long: `A tool to batch download files that were attached to slack. For example:

Please pre-extract the archive file downloaded from the Slack console.
Pass the path to the extracted directory as an argument.
(By default, the files downloaded in batches will be placed in the directory passed as the argument.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("Specify the extracted directory exported from slack.")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		path := strings.Join(args, "")
		sfd.Run(path)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
