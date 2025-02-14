// Package cmd contains the root command for the xkcd CLI.
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jucrouzet/xkcd/internal/pkg/cli"
	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

var (
	apiClient     *xkcd.Client
	contextCancel context.CancelFunc
	ctx           context.Context
	index         *cli.Index
	logger        *slog.Logger
	outToClose    io.Closer

	build             = "development"
	indexPath         = ""
	json              = false
	noColor           = false
	outIsATTY         = false
	outputContentType = "text/plain"
	outputVal         = "stdout"
	timeout           = uint32(30)
	verbose           = false
	version           = "0.0.0"
)

var rootCmd = &cobra.Command{
	Use:   "xkcd",
	Short: "xkcd in your terminal",
	Long:  `Get your daily dose of xkcd comic, search for a post, or browse them, right from the terminal.`,
	Run: func(cmd *cobra.Command, _ []string) {
		checkErr(cmd.Help(), cmd)
		os.Exit(0)
	},
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		ctx, contextCancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		cmd.SetContext(ctx)

		if !setOut(cmd) {
			return
		}
		logger = getLogger(cmd)
		apiClient = xkcd.New(
			xkcd.WithLogger(logger),
		)
		if noColor {
			color.NoColor = true
		}

		var err error
		index, err = cli.NewIndex(indexPath, logger)
		checkErr(err, cmd, "failed to open index")
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if err := index.Close(); err != nil {
			logger.Warn("failed to close index", slog.String("error", err.Error()))
		}
		if outToClose != nil {
			if err := outToClose.Close(); err != nil {
				logger.Warn("failed to close output", slog.String("error", err.Error()))
			}
		}
		if contextCancel != nil {
			contextCancel()
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

type httpOut struct {
	buffer  *bytes.Buffer
	closed  *uint32
	command *cobra.Command
}

func newHTTPOut(cmd *cobra.Command) *httpOut {
	return &httpOut{
		buffer:  &bytes.Buffer{},
		closed:  new(uint32),
		command: cmd,
	}
}

func (h *httpOut) Read(p []byte) (int, error) {
	return h.buffer.Read(p)
}

func (h *httpOut) Write(p []byte) (int, error) {
	if atomic.LoadUint32(h.closed) == 1 {
		return 0, io.ErrClosedPipe
	}
	return h.buffer.Write(p)
}

func (h *httpOut) Close() error {
	start := time.Now()
	log := logger.With("output", "url").With("output_url", outputVal)
	atomic.StoreUint32(h.closed, 1)
	req, err := http.NewRequestWithContext(h.command.Context(), http.MethodPost, outputVal, h.buffer)
	if err != nil {
		return fmt.Errorf("failed to create output request: %w", err)
	}
	if json {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", outputContentType)
	}
	req.Header.Add("User-Agent", "xkcd-cli/"+version)

	log.Debug("making output request")
	resp, err := http.DefaultClient.Do(req)
	log.Debug("ended output request")
	if err != nil {
		logger.Warn("failed to send output request", slog.String("error", err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		logger.Warn("output request failed", slog.Int("status", resp.StatusCode))
	}
	log.Debug("finished sending output request", slog.Duration("duration", time.Since(start)))
	return nil
}

func setOut(cmd *cobra.Command) bool {
	outputVal = strings.ToLower(strings.TrimSpace(outputVal))
	switch outputVal {
	case "stdout", "-":
		cmd.SetOut(os.Stdout)
		outIsATTY = term.IsTerminal(int(os.Stdout.Fd()))
		return true
	case "stderr":
		cmd.SetOut(os.Stdout)
		return true
	}
	if strings.HasPrefix(outputVal, "http://") || strings.HasPrefix(outputVal, "https://") {
		w := newHTTPOut(cmd)
		cmd.SetOut(w)
		outToClose = w
		return true
	}
	f, err := os.OpenFile(outputVal, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fatal(cmd, fmt.Sprintf("failed to open output file: %v", err))
		return false
	}
	cmd.SetOut(f)
	outToClose = f
	return true
}

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	indexPath = path.Join(home, ".xkcd.index")

	rootCmd.PersistentFlags().StringVar(&indexPath, "index", indexPath, "Path to the index file")
	rootCmd.PersistentFlags().BoolVarP(&json, "json", "j", false, "use the json format for logging and output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "do not use color in output even if terminal supports it")
	rootCmd.PersistentFlags().StringVarP(&outputVal, "output", "o", "stdout", "output of the cli, can be 'stdout', 'stderr', a file path to be appended on or an url to POST on")
	rootCmd.PersistentFlags().Uint32VarP(&timeout, "timeout", "t", 30000, "timeout in milliseconds")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging mode")
}
