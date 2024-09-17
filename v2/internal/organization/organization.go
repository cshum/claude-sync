package organization

import (
	"fmt"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/internal/provider"
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "organization",
		Usage: "Manage AI organizations",
		Subcommands: []*cli.Command{
			{
				Name:   "ls",
				Usage:  "List all available organizations with required capabilities",
				Action: listOrganizations,
			},
			{
				Name:  "set",
				Usage: "Set the active organization",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "org-id",
						Usage: "ID of the organization to set as active",
					},
					&cli.StringFlag{
						Name:  "provider",
						Value: "claude.ai",
						Usage: "Specify the provider for repositories without .claudesync",
					},
				},
				Action: setOrganization,
			},
		},
	}
}

func listOrganizations(c *cli.Context) error {
	cfg := config.GetConfig()
	prov, err := provider.GetProvider(cfg.Get("active_provider").(string), cfg)
	if err != nil {
		return err
	}

	organizations, err := prov.GetOrganizations()
	if err != nil {
		return err
	}

	if len(organizations) == 0 {
		fmt.Println("No organizations with required capabilities (chat and claude_pro) found.")
		return nil
	}

	fmt.Println("Available organizations with required capabilities:")
	for i, org := range organizations {
		fmt.Printf("  %d. %s (ID: %s)\n", i+1, org.Name, org.ID)
	}

	return nil
}

func setOrganization(c *cli.Context) error {
	cfg := config.GetConfig()
	providerName := c.String("provider")
	cfg.Set("active_provider", providerName, true)

	prov, err := provider.GetProvider(providerName, cfg)
	if err != nil {
		return err
	}

	organizations, err := prov.GetOrganizations()
	if err != nil {
		return err
	}

	var selectedOrg provider.Organization
	orgID := c.String("org-id")
	if orgID != "" {
		for _, org := range organizations {
			if org.ID == orgID {
				selectedOrg = org
				break
			}
		}
		if selectedOrg.ID == "" {
			return fmt.Errorf("organization with ID %s not found", orgID)
		}
	} else {
		fmt.Println("Available organizations:")
		for i, org := range organizations {
			fmt.Printf("  %d. %s (ID: %s)\n", i+1, org.Name, org.ID)
		}
		var selection int
		fmt.Print("Enter the number of the organization you want to work with: ")
		_, err := fmt.Scanf("%d", &selection)
		if err != nil || selection < 1 || selection > len(organizations) {
			return fmt.Errorf("invalid selection")
		}
		selectedOrg = organizations[selection-1]
	}

	cfg.Set("active_organization_id", selectedOrg.ID, true)
	fmt.Printf("Selected organization: %s (ID: %s)\n", selectedOrg.Name, selectedOrg.ID)

	// Clear project-related settings when changing organization
	cfg.Set("active_project_id", nil, true)
	cfg.Set("active_project_name", nil, true)
	fmt.Println("Project settings cleared. Please select or create a new project for this organization.")

	return nil
}
