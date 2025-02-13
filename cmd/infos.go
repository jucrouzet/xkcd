package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jucrouzet/xkcd/internal/pkg/cli"
	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

var infosCmd = &cobra.Command{
	Use:   `infos ["latest"|number]`,
	Short: "Get infos of a xkcd post",
	Long: `Retrieve informations of a xkcd post, by giving its number or 'latest' to get the latest one.
If no argument is provided, it defaults to the latest post.`,
	Aliases: []string{"i"},
	Args:    cobra.MaximumNArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		valids := []string{"", "latest"}

		return valids, cobra.ShellCompDirectiveNoFileComp
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
		checkErr(err, cmd, "failed to get post")
		checkErr(cli.DisplayPostInfos(cmd.OutOrStdout(), post, json), cmd, "failed to display post informations")
	},
}

func init() {
	rootCmd.AddCommand(infosCmd)
}
