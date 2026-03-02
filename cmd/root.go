package cmd

import (
	"fmt"
	"os"

	"github.com/benfourie/fl/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fl",
	Short: "flow — your Jira + Git + Calendar CLI",
	Long: `fl keeps you in the flow.

Manage Jira tickets, create branches, log notes,
and see today's work — all without leaving the terminal.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(config.Init)

	rootCmd.AddCommand(branchCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(moveCmd)
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(icalCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
}
