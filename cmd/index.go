package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index operation",
	Short: "Index operations",
	Long: `In order to search posts, xkcd maintains an index of all available posts.
Index commands allow you to manipulate this index.`,
}

func checkIndexInitialized(cmd *cobra.Command) {
	if index.Initized() {
		return
	}
	fatal(
		cmd,
		fmt.Sprintf(
			"index is not initialized, run '%s index init' first",
			os.Args[0],
		),
	)
}

func checkErrIndex(err error, cmd *cobra.Command, message ...string) {
	if err == nil {
		return
	}
	logMsg := fmt.Sprintf(
		"a fatal error occcured with index, you should run '%s index init -f' to reintialize it",
		os.Args[0],
	)
	if len(message) > 0 {
		logMsg = fmt.Sprintf(
			"a fatal error occcured with index: %s, you should run '%s index init -f' to reintialize it",
			message[0],
			os.Args[0],
		)
	}
	logger.Warn("fatal error occurred with index", slog.Any("error", err))
	fatal(cmd, logMsg)
}

func init() {
	rootCmd.AddCommand(indexCmd)
}
