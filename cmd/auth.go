package cmd

import (
	"github.com/benfourie/fl/internal/calendar"
	"github.com/benfourie/fl/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Jira and calendar providers",
}

var authJiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Save Jira credentials to the OS keychain",
	RunE:  runAuthJira,
}

var authGoogleCmd = &cobra.Command{
	Use:   "google",
	Short: "Authenticate with Google Calendar via OAuth",
	RunE:  runAuthGoogle,
}

var authOutlookCmd = &cobra.Command{
	Use:   "outlook",
	Short: "Authenticate with Outlook/MS 365 via OAuth",
	RunE:  runAuthOutlook,
}

func init() {
	authCmd.AddCommand(authJiraCmd)
	authCmd.AddCommand(authGoogleCmd)
	authCmd.AddCommand(authOutlookCmd)
}

func runAuthJira(cmd *cobra.Command, args []string) error {
	return config.SetupJiraAuth()
}

func runAuthGoogle(cmd *cobra.Command, args []string) error {
	return calendar.SetupGoogleAuth()
}

func runAuthOutlook(cmd *cobra.Command, args []string) error {
	return calendar.SetupOutlookAuth()
}
