package cli

import (
	"fmt"
	"os"

	"github.com/mikeocool/tape/core"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stops a running dev environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envName := args[0]

		// Get box summary to check the state
		summary, err := core.GetBoxSummary(envName)
		if err != nil {
			fmt.Printf("Error getting box summary for %s: %v\n", envName, err)
			os.Exit(1)
		}

		// Check if the box is running
		if summary.State != core.BoxStateRunning {
			fmt.Printf("Cannot remove %s: container is not running (current state: %s)\n", envName, summary.State)
			os.Exit(1)
		}

		fmt.Printf("Stopping container %s...\n", envName)

		// Stop the container
		err = core.StopContainer(summary.ContainerID)
		if err != nil {
			fmt.Printf("Error stopping container: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully stopped and removed container for %s\n", envName)
	},
}
