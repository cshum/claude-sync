package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cshum/claude-sync/v2/internal/auth"
	"github.com/cshum/claude-sync/v2/internal/chat"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/internal/organization"
	"github.com/cshum/claude-sync/v2/internal/project"
	"github.com/cshum/claude-sync/v2/internal/sync"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "claudesync",
		Usage: "Synchronize local files with Claude.ai projects",
		Commands: []*cli.Command{
			auth.Command(),
			chat.Command(),
			config.Command(),
			organization.Command(),
			project.Command(),
			sync.Command(),
			{
				Name:  "upgrade",
				Usage: "Upgrade ClaudeSync to the latest version",
				Action: func(c *cli.Context) error {
					fmt.Println("Upgrading ClaudeSync...")
					// Implement upgrade logic here
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
