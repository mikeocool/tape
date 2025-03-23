package cli

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// rootCmd.AddCommand(versionCmd)

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(rmCmd)
}
