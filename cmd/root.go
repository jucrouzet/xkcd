// Package cmd contains the root command for the xkcd CLI.
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/fatih/color"
	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"

	"github.com/jucrouzet/xkcd/internal/pkg/cli"
	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

var (
	VERSION = "0.0.0"
	BUILD   = "development"
)

var (
	apiClient     *xkcd.Client
	contextCancel context.CancelFunc
	ctx           context.Context
	index         *cli.Index
	indexPath     = ""
	json          = false
	logger        *slog.Logger
	noColor       = false
	timeout       = uint32(30)
	verbose       = false
)

var rootCmd = &cobra.Command{
	Use:   "xkcd",
	Short: "xkcd in your terminal",
	Long:  `Get your daily dose of xkcd comics or search an image, right from the terminal.`,
	Run: func(cmd *cobra.Command, _ []string) {
		checkErr(cmd.Help(), cmd)
		os.Exit(0)
	},
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		logger = getLogger(cmd)
		apiClient = xkcd.New(
			xkcd.WithLogger(logger),
		)
		ctx, contextCancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		cmd.SetContext(ctx)
		if noColor {
			color.NoColor = true
		}

		var err error
		index, err = cli.NewIndex(indexPath, logger)
		checkErr(err, cmd, "failed to open index")
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if contextCancel != nil {
			contextCancel()
		}
		if err := index.Close(); err != nil {
			logger.Warn("failed to close index", slog.String("error", err.Error()))
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func checkErr(err error, cmd *cobra.Command, message ...string) {
	if err == nil {
		return
	}
	logMsg := "a fatal error occcured"
	if len(message) > 0 {
		logMsg = message[0]
	}
	logger.Warn("fatal error", slog.Any("error", err))
	fatal(cmd, logMsg)
}

func fatal(cmd *cobra.Command, message string) {
	if json {
		logger.Error(message)
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), color.RedString("*** %s", message))
	}
	os.Exit(1)
}

func getLogger(cmd *cobra.Command) *slog.Logger {
	lvl := slog.LevelInfo

	if verbose {
		lvl = slog.LevelDebug
	}

	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{
			Level: lvl,
		})
	} else {
		handler = tint.NewHandler(cmd.ErrOrStderr(), &tint.Options{
			Level:   lvl,
			NoColor: noColor,
		})
	}
	return slog.New(handler)
}

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	indexPath = path.Join(home, ".xkcd.index")

	rootCmd.PersistentFlags().BoolVarP(&json, "json", "j", false, "json output format for logging and output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging mode")
	rootCmd.PersistentFlags().Uint32VarP(&timeout, "timeout", "t", 30000, "timeout in milliseconds")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "do not use color in output even if terminal supports it")
	rootCmd.PersistentFlags().StringVar(&indexPath, "index", indexPath, "Path to the index file")
}
