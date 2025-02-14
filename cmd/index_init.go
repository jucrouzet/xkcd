package cmd

import (
	"github.com/spf13/cobra"
)

var (
	indexInitCmdForce   = false
	indexInitCmdOffline = false
)

var indexInitCmd = &cobra.Command{
	Use:   `init`,
	Short: "Initialize the index",
	Run: func(cmd *cobra.Command, _ []string) {
		if index.Initized() && !indexInitCmdForce {
			fatal(cmd, "index is already initialized")
			return
		}
		checkErr(index.Init(cmd.Context(), indexInitCmdForce, indexInitCmdOffline), cmd, "failed to initialize index")
	},
}

func init() {
	indexInitCmd.Flags().BoolVarP(&indexInitCmdForce, "force", "f", false, "force reinitialization of the index (all previous data is lost)")
	indexInitCmd.Flags().BoolVar(&indexInitCmdOffline, "offline", false, "initialize the index with offline mode, image content will be stored in index for offline")
	indexCmd.AddCommand(indexInitCmd)
}
