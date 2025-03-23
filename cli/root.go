package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "boxcut",
	Short: "Manage dev environments",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("boxcut")
	},
}
