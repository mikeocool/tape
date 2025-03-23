package cli

import (
	"fmt"
	"os"

	"github.com/mikeocool/tape/core"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm [name]",
	Short: "Remove a stopped container",
	Long:  `Remove a container for the specified environment name if it is in stopped state.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envName := args[0]

		// Get box summary to check container state
		summary, err := core.GetBoxSummary(envName)
		if err != nil {
			fmt.Printf("Error getting box summary for %s: %v\n", envName, err)
			os.Exit(1)
		}

		// Check if the container is in stopped state
		if summary.State != core.BoxStateStopped {
			fmt.Printf("Cannot remove %s: container is not stopped (current state: %s)\n", envName, summary.State)
			os.Exit(1)
		}

		fmt.Printf("Removing container %s...\n", envName)

		// Remove the container
		err = core.RemoveContainer(summary.ContainerID)
		if err != nil {
			fmt.Printf("Error removing container: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully removed container for %s\n", envName)
	},
}
