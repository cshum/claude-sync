package chat

import (
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "chat",
		Usage: "Manage and synchronize chats",
		Subcommands: []*cli.Command{
			{
				Name:   "pull",
				Usage:  "Synchronize chats and their artifacts from the remote source",
				Action: pullAction,
			},
			{
				Name:   "ls",
				Usage:  "List all chats",
				Action: listAction,
			},
			{
				Name:  "rm",
				Usage: "Delete chat conversations",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "all",
						Usage: "Delete all chats",
					},
				},
				Action: removeAction,
			},
			{
				Name:  "init",
				Usage: "Initializes a new chat conversation",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "name",
						Usage: "Name of the chat conversation",
					},
					&cli.StringFlag{
						Name:  "project",
						Usage: "UUID of the project to associate the chat with",
					},
				},
				Action: initAction,
			},
			{
				Name:  "message",
				Usage: "Send a message to a specified chat or create a new chat and send the message",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "chat",
						Usage: "UUID of the chat to send the message to",
					},
					&cli.StringFlag{
						Name:  "timezone",
						Value: "UTC",
						Usage: "Timezone for the message",
					},
				},
				Action: messageAction,
			},
		},
	}
}

func pullAction(c *cli.Context) error {
	// Implement pull action
	return nil
}

func listAction(c *cli.Context) error {
	// Implement list action
	return nil
}

func removeAction(c *cli.Context) error {
	// Implement remove action
	return nil
}

func initAction(c *cli.Context) error {
	// Implement init action
	return nil
}

func messageAction(c *cli.Context) error {
	// Implement message action
	return nil
}
