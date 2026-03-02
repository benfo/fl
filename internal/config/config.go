package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	appName        = "fl"
	keyringService = "fl-cli"
)

// Init is called by cobra on startup. It loads config from ~/.fl/config.yaml.
func Init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot find home directory:", err)
		return
	}

	cfgDir := filepath.Join(home, ".fl")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(cfgDir)

	// Env overrides: FL_JIRA_HOST, FL_JIRA_EMAIL, etc.
	viper.SetEnvPrefix("FL")
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "warning: config error:", err)
		}
	}

	mergeRepoConfig()
}

// mergeRepoConfig looks for a .fl.yaml in the current git repo root and
// merges it on top of the global config. Repo-level values take precedence.
// Silently no-ops when not in a git repo or when .fl.yaml is absent.
func mergeRepoConfig() {
	root, err := GitRoot()
	if err != nil {
		return // not in a git repo
	}

	repoConfigPath := filepath.Join(root, ".fl.yaml")
	if _, err := os.Stat(repoConfigPath); err != nil {
		return // no repo-level config
	}

	// Use a separate Viper instance so we never alter the global write path
	// (WriteConfigAs in auth commands must keep targeting ~/.fl/config.yaml).
	rv := viper.New()
	rv.SetConfigFile(repoConfigPath)
	if err := rv.ReadInConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: repo config error:", err)
		return
	}

	if err := viper.MergeConfigMap(rv.AllSettings()); err != nil {
		fmt.Fprintln(os.Stderr, "warning: merging repo config:", err)
	}
}

// GitRoot returns the root directory of the current git repository.
func GitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func setDefaults() {
	viper.SetDefault("tracker.provider", "jira")
	viper.SetDefault("branch.template", "{{.Type}}/{{.Key}}-{{.Title}}")
	viper.SetDefault("calendar.providers", []string{})
}

// TrackerProvider returns the configured tracker backend ("jira" or "trello").
func TrackerProvider() string {
	return viper.GetString("tracker.provider")
}

// JiraHost returns the configured Jira base URL.
func JiraHost() string {
	return viper.GetString("jira.host")
}

// JiraEmail returns the configured Jira user email.
func JiraEmail() string {
	return viper.GetString("jira.email")
}

// JiraToken retrieves the Jira API token from the OS keychain.
func JiraToken() (string, error) {
	email := JiraEmail()
	if email == "" {
		return "", fmt.Errorf("jira.email not configured — run: fl auth jira")
	}
	token, err := keyring.Get(keyringService, email)
	if err != nil {
		return "", fmt.Errorf("no token in keychain for %s — run: fl auth jira", email)
	}
	return token, nil
}

// SetupJiraAuth prompts for Jira credentials and stores the token in the keychain.
func SetupJiraAuth() error {
	var host, email, token string

	fmt.Print("Jira host (e.g. https://company.atlassian.net): ")
	fmt.Scanln(&host)

	fmt.Print("Jira email: ")
	fmt.Scanln(&email)

	fmt.Print("Jira API token (https://id.atlassian.com/manage-profile/security/api-tokens): ")
	fmt.Scanln(&token)

	if err := keyring.Set(keyringService, email, token); err != nil {
		return fmt.Errorf("saving token to keychain: %w", err)
	}

	// Write host + email to config file.
	viper.Set("jira.host", host)
	viper.Set("jira.email", email)

	cfgDir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		return err
	}

	cfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := viper.WriteConfigAs(cfgPath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Credentials saved. Config written to %s\n", cfgPath)
	return nil
}

// JiraProjects returns the list of project keys to filter by, e.g. ["PROJ", "DEV"].
// Returns nil when no projects are configured, meaning all projects are searched.
func JiraProjects() []string {
	return viper.GetStringSlice("jira.projects")
}

// BranchTemplate returns the configured branch name template.
func BranchTemplate() string {
	return viper.GetString("branch.template")
}

// CalendarProviders returns the list of enabled calendar providers.
func CalendarProviders() []string {
	return viper.GetStringSlice("calendar.providers")
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fl"), nil
}
