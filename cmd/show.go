package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jucrouzet/xkcd/internal/pkg/cli"
	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

var (
	displayer cli.Displayer
	showInfos = false
)

var showCmd = &cobra.Command{
	Use:   `show ["latest"|number]`,
	Short: "Shows a xkcd post",
	Long: `Shows a xkcd post, by giving its number or 'latest' to get the latest one.
If no argument is provided, it defaults to the latest post.`,
	Aliases: []string{"s", "display"},
	Args:    cobra.MaximumNArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		valids := []string{"", "latest"}

		return valids, cobra.ShellCompDirectiveNoFileComp
	},
	PreRun: func(cmd *cobra.Command, _ []string) {
		if json {
			fatal(cmd, "cannot display images in json mode")
			return
		}
		displayer = cli.GetDisplayer(logger)
		if displayer == nil {
			fatal(cmd, "terminal does not support image display")
			return
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var post *xkcd.Post
		var err error
		if len(args) == 0 || args[0] == "latest" {
			post, err = apiClient.GetLatest(cmd.Context())
		} else {
			n, errParse := strconv.Atoi(args[0])
			checkErr(errParse, cmd, "invalid post number")
			post, err = apiClient.GetPost(cmd.Context(), n)
		}
		checkErr(err, cmd, "failed to get latest post")

		if showInfos {
			checkErr(cli.DisplayPostInfos(cmd.OutOrStdout(), post, json), cmd, "failed to display post informations")
		}

		checkErr(cli.DisplayPostImage(cmd.Context(), cmd.OutOrStdout(), post, displayer, logger), cmd, "failed to display post")
	},
}

func init() {
	showCmd.Flags().BoolVarP(&showInfos, "infos", "i", false, "show post informations")
	rootCmd.AddCommand(showCmd)
}
