// Package cmd contains the version command for the xkcd CLI.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	versionCmdFull = false
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "get software version",
	Run: func(cmd *cobra.Command, _ []string) {
		if versionCmdFull {
			fmt.Fprintf(cmd.OutOrStderr(), "%s-%s\n", VERSION, BUILD)
		} else {
			fmt.Fprintf(cmd.OutOrStderr(), "%s\n", VERSION)
		}
		os.Exit(0)
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionCmdFull, "full", "f", false, "show full version including commit hash")
	rootCmd.AddCommand(versionCmd)
}
