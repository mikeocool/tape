package cli

import (
	"fmt"
	"os"

	"github.com/mikeocool/tape/core"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List environments",
	Run: func(cmd *cobra.Command, args []string) {
		envs, err := core.ListBoxConfigs()
		if err != nil {
			fmt.Printf("Error listing environments: %v\n", err)
			os.Exit(1)
		}

		// Find the longest environment name for proper alignment
		maxNameLength := 0
		for _, name := range envs {
			if len(name) > maxNameLength {
				maxNameLength = len(name)
			}
		}

		// Format string with fixed width for the first column
		formatStr := fmt.Sprintf("%%-%ds\t%%s\n", maxNameLength)
		errorFormatStr := fmt.Sprintf("%%-%ds\terror\t%%s\n", maxNameLength)

		for _, name := range envs {
			summary, err := core.GetBoxSummary(name)
			if err != nil {
				fmt.Printf(errorFormatStr, name, err)
				continue
			}

			fmt.Printf(formatStr, name, summary.State)
		}
	},
}
