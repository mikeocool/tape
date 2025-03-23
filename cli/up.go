package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mikeocool/boxd/boxcut/core"
	"github.com/spf13/cobra"
)

var (
	rebuildFlag bool
)

var upCmd = &cobra.Command{
	Use:   "up [name]",
	Short: "Starts a dev environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envName := args[0]
		fmt.Println("Starting box", envName)

		// Load the configuration
		config, err := core.LoadBoxConfig(envName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Create additional arguments if rebuild flag is set
		additionalArgs := []string{}
		if rebuildFlag {
			additionalArgs = append(additionalArgs,
				"--build-no-cache",
				"--remove-existing-container")
		}

		// Create and execute the devcontainer command
		devCmd := core.DevcontainerCommand{
			BoxConfig:      *config,
			Command:        "up",
			AdditionalArgs: additionalArgs,
		}

		err = devCmd.Execute()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Printf("Error executing command: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	upCmd.Flags().BoolVar(&rebuildFlag, "rebuild", false, "Rebuild the container with no cache and remove existing container")
}
