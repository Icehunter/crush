package cmd

import (
	"fmt"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout [platform]",
	Short: "Logout Crush from a platform",
	Long: `Logout Crush from a specified platform.
The platform should be provided as an argument.
Available platforms are: hyper, copilot, claude, anthropic.
If no platform is specified, all platforms are logged out.`,
	Example: `
# Logout from all platforms
crush logout

# Logout from GitHub Copilot
crush logout copilot

# Logout from Claude
crush logout claude

# Logout from Hyper
crush logout hyper
  `,
	ValidArgs: []cobra.Completion{
		"hyper",
		"copilot",
		"github",
		"github-copilot",
		"claude",
		"claude-code",
		"anthropic",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := setupAppWithProgressBar(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		cfg := app.Store()

		if len(args) == 0 {
			errs := []error{
				logoutProvider(cfg, "hyper", "Hyper"),
				logoutProvider(cfg, "copilot", "GitHub Copilot"),
				logoutProvider(cfg, "anthropic", "Claude"),
			}
			logged := false
			for _, err := range errs {
				if err != nil {
					fmt.Println("Warning:", err)
				} else {
					logged = true
				}
			}
			if logged {
				fmt.Println("Logged out successfully.")
			}
			return nil
		}

		switch args[0] {
		case "hyper":
			return logoutProvider(cfg, "hyper", "Hyper")
		case "copilot", "github", "github-copilot":
			return logoutProvider(cfg, "copilot", "GitHub Copilot")
		case "claude", "claude-code", "anthropic":
			return logoutProvider(cfg, "anthropic", "Claude")
		default:
			return fmt.Errorf("unknown platform: %s", args[0])
		}
	},
}

func logoutProvider(cfg *config.ConfigStore, providerKey, displayName string) error {
	apiKeyField := "providers." + providerKey + ".api_key"
	oauthField := "providers." + providerKey + ".oauth"

	if !cfg.HasConfigField(config.ScopeGlobal, apiKeyField) && !cfg.HasConfigField(config.ScopeGlobal, oauthField) {
		fmt.Printf("Not logged in to %s.\n", displayName)
		return nil
	}

	if cfg.HasConfigField(config.ScopeGlobal, apiKeyField) {
		if err := cfg.RemoveConfigField(config.ScopeGlobal, apiKeyField); err != nil {
			return fmt.Errorf("failed to remove %s API key: %w", displayName, err)
		}
	}
	if cfg.HasConfigField(config.ScopeGlobal, oauthField) {
		if err := cfg.RemoveConfigField(config.ScopeGlobal, oauthField); err != nil {
			return fmt.Errorf("failed to remove %s OAuth token: %w", displayName, err)
		}
	}

	fmt.Printf("Logged out of %s.\n", displayName)
	return nil
}
