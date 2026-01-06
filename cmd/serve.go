package cmd

import "github.com/spf13/cobra"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "运行 ispace 服务",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags()
	},
}
