package auth

import (
	"bufio"
	"fmt"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/urfave/cli/v2"
	"os"
	"strings"
	"time"
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
	cfg := config.GetConfig()
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Choose provider (claude.ai) [claude.ai]: ")
	providerName, _ := reader.ReadString('\n')
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "claude.ai"
	}

	fmt.Println("A session key is required to call: https://api.claude.ai/api")
	fmt.Println("To obtain your session key, please follow these steps:")
	fmt.Println("1. Open your web browser and go to https://claude.ai")
	fmt.Println("2. Log in to your Claude account if you haven't already")
	fmt.Println("3. Once logged in, open your browser's developer tools:")
	fmt.Println("   - Chrome/Edge: Press F12 or Ctrl+Shift+I (Cmd+Option+I on Mac)")
	fmt.Println("   - Firefox: Press F12 or Ctrl+Shift+I (Cmd+Option+I on Mac)")
	fmt.Println("   - Safari: Enable developer tools in Preferences > Advanced, then press Cmd+Option+I")
	fmt.Println("4. In the developer tools, go to the 'Application' tab (Chrome/Edge) or 'Storage' tab (Firefox)")
	fmt.Println("5. In the left sidebar, expand 'Cookies' and select 'https://claude.ai'")
	fmt.Println("6. Locate the cookie named 'sessionKey' and copy its value. Ensure that the value is not URL-encoded.")

	fmt.Print("Please enter your sessionKey: ")
	sessionKey, _ := reader.ReadString('\n')
	sessionKey = strings.TrimSpace(sessionKey)

	defaultExpiry := time.Now().AddDate(0, 1, 0) // Default to 1 month from now
	fmt.Printf("Please enter the expires time for the sessionKey (optional) [%s]: ", defaultExpiry.Format(time.RFC1123))
	expiryStr, _ := reader.ReadString('\n')
	expiryStr = strings.TrimSpace(expiryStr)

	var expiry time.Time
	var err error
	if expiryStr == "" {
		expiry = defaultExpiry
	} else {
		expiry, err = time.Parse(time.RFC1123, expiryStr)
		if err != nil {
			return fmt.Errorf("invalid expiry time format: %v", err)
		}
	}

	err = cfg.SetSessionKey(providerName, sessionKey, expiry)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully stored session key for %s. Session key stored globally.\n", providerName)
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
