package cmd

import (
	"log/slog"
	"time"

	"github.com/spf13/cobra"
)

var (
	indexUpdateCmdCheck   = false
	indexUpdateCmdForce   = false
	indexUpdateCmdWorkers = uint(5)
)

var indexUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update if index should be updated, and if so, update it",
	Run: func(cmd *cobra.Command, _ []string) {
		checkIndexInitialized(cmd)
		lastDate, lastNum, err := index.GetLastUpdate()
		checkErrIndex(err, cmd, "failed to get last update")
		if lastDate.IsZero() {
			logger = logger.With(slog.String("last_update", "never"))
		} else {
			logger = logger.With(slog.Time("last_update", lastDate))
		}
		if !indexUpdateCmdForce && time.Since(lastDate).Hours() < 24 {
			logger.Debug("index is up to date")
			return
		}
		if indexUpdateCmdCheck {
			fatal(cmd, "index is outdated")
			return
		}
		logger.Debug("updating index")
		latest, err := apiClient.GetLatest(cmd.Context())
		checkErrIndex(err, cmd, "failed to get latest post")
		if latest.Num <= lastNum {
			logger.Debug("index is up to date")
			return
		}
		checkErr(index.Update(cmd.Context(), apiClient, lastNum+1, latest.Num, indexUpdateCmdWorkers), cmd, "failed to update index")
	},
}

func init() {
	indexUpdateCmd.Flags().BoolVarP(&indexUpdateCmdCheck, "check", "c", false, "only check if index should be updated, do not update it")
	indexUpdateCmd.Flags().BoolVarP(&indexUpdateCmdForce, "force", "f", false, "force update of the index even if it is up to date")
	indexUpdateCmd.Flags().UintVarP(&indexUpdateCmdWorkers, "workers", "w", 5, "how many posts should we process concurrently")
	indexCmd.AddCommand(indexUpdateCmd)
}
