package cli

import (
	"github.com/mikeocool/tape/ssh"
	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "SSH into dev environment",
	Run: func(cmd *cobra.Command, args []string) {
		ssh.Start()
	},
}
