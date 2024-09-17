package auth

import (
	"fmt"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/internal/provider"
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Manage authentication",
		Subcommands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Authenticate with an AI provider",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "provider",
						Aliases: []string{"p"},
						Value:   "claude.ai",
						Usage:   "The provider to use for this project",
					},
				},
				Action: login,
			},
			{
				Name:   "logout",
				Usage:  "Log out from all AI providers",
				Action: logout,
			},
			{
				Name:   "ls",
				Usage:  "List all authenticated providers",
				Action: list,
			},
		},
	}
}

func login(c *cli.Context) error {
	providerName := c.String("provider")
	cfg := config.GetConfig()
	prov, err := provider.GetProvider(providerName, cfg)
	if err != nil {
		return err
	}

	sessionKey, expiry, err := prov.Login()
	if err != nil {
		return err
	}

	err = cfg.SetSessionKey(providerName, sessionKey, expiry)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully authenticated with %s. Session key stored globally.\n", providerName)
	return nil
}

func logout(c *cli.Context) error {
	cfg := config.GetConfig()
	cfg.ClearAllSessionKeys() // Implement this method in the Config struct
	fmt.Println("Logged out from all providers successfully.")
	return nil
}

func list(c *cli.Context) error {
	cfg := config.GetConfig()
	providers := cfg.GetProvidersWithSessionKeys() // Implement this method in the Config struct
	if len(providers) == 0 {
		fmt.Println("No authenticated providers found.")
	} else {
		fmt.Println("Authenticated providers:")
		for _, p := range providers {
			fmt.Printf("  - %s\n", p)
		}
	}
	return nil
}
