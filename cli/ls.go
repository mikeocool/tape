package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List environments",
	Run: func(cmd *cobra.Command, args []string) {
		files, err := filepath.Glob("sample-config/*.yml")
		if err != nil {
			fmt.Printf("Error reading config files: %v\n", err)
			return
		}

		for _, file := range files {
			// Get just the filename without path and extension
			base := filepath.Base(file)
			name := strings.TrimSuffix(base, ".yml")
			fmt.Println(name)

			// TODO report if container is running or not
		}
	},
}
