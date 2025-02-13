package cmd

import (
	"github.com/spf13/cobra"
)

var (
	indexInitCmdForce = false
)

var indexInitCmd = &cobra.Command{
	Use:   `init`,
	Short: "Initialize the index",
	Run: func(cmd *cobra.Command, _ []string) {
		if index.Initized() && !indexInitCmdForce {
			fatal(cmd, "index is already initialized")
			return
		}
		checkErr(index.Init(indexInitCmdForce), cmd, "failed to initialize index")
	},
}

func init() {
	indexInitCmd.Flags().BoolVarP(&indexInitCmdForce, "force", "f", false, "force reinitialization of the index (all previous data is lost)")
	indexCmd.AddCommand(indexInitCmd)
}
