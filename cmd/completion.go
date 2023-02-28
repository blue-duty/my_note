package cmd

import "github.com/spf13/cobra"

var completionCmd = &cobra.Command{
	Use:    "completion",
	Short:  "Generate completion script",
	Hidden: true,
}
