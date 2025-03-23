package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mikeocool/boxd/boxcut/core"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [envName] [cmd] [args...]",
	Short: "Execute a command in a dev environment",
	Long: `Execute a command inside a dev environment.
Example: boxcut exec myenv ls -la
Everything after -- will be passed directly to the container.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Error: Missing environment name")
			cmd.Usage()
			os.Exit(1)
		}

		// Get environment name
		envName := args[0]

		// Everything after name is the command and its arguments
		execArgs := args[1:]
		if len(execArgs) < 1 {
			fmt.Println("Error: No command specified to execute")
			cmd.Usage()
			os.Exit(1)
		}

		// TODO look at https://stackoverflow.com/questions/72708535/cobra-cli-pass-all-arguments-and-flags-to-an-executable
		// to fix args passing through

		// Load the configuration
		config, err := core.LoadBoxConfig(envName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Create and execute the devcontainer command
		devCmd := core.DevcontainerCommand{
			BoxConfig:      *config,
			Command:        "exec",
			AdditionalArgs: execArgs,
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
