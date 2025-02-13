package cmd

import (
	"log/slog"
	"time"

	"github.com/spf13/cobra"
)

var (
	indexUpdateCmdOnly  = false
	indexUpdateCmdForce = false
)

var indexUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update if index should be updated, and if so, update it",
	Run: func(cmd *cobra.Command, _ []string) {
		checkIndexInitialized(cmd)
		last, err := index.GetLastUpdate()
		checkErrIndex(err, cmd, "failed to update last update")
		if last.IsZero() {
			logger = logger.With(slog.String("last_update", "never"))
		} else {
			logger = logger.With(slog.Time("last_update", last))
		}
		if !indexUpdateCmdForce && time.Since(last).Hours() < 24 {
			logger.Debug("index is up to date")
			return
		}
		if indexUpdateCmdOnly {
			fatal(cmd, "index is outdated")
			return
		}
		logger.Debug("updating index")
	},
}

func init() {
	indexUpdateCmd.Flags().BoolVarP(&indexUpdateCmdOnly, "only-update", "o", false, "only update if index should be updated, do not update it")
	indexUpdateCmd.Flags().BoolVarP(&indexUpdateCmdForce, "force", "f", false, "force update of the index even if it is up to date")
	indexCmd.AddCommand(indexUpdateCmd)
}
