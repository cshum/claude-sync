package project

import (
	"fmt"
	"github.com/cshum/claude-sync/v2/providerapi"
	"os"
	"path/filepath"

	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/internal/provider"
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "project",
		Usage: "Manage AI projects within the active organization",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Creates a new project for the selected provider",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "name",
						Usage: "The name of the project",
					},
					&cli.StringFlag{
						Name:  "description",
						Value: "Project created with ClaudeSync",
						Usage: "The project description",
					},
					&cli.StringFlag{
						Name:  "local-path",
						Value: ".",
						Usage: "The local path for the project",
					},
					&cli.StringFlag{
						Name:  "provider",
						Value: "claude.ai",
						Usage: "The provider to use for this project",
					},
					&cli.StringFlag{
						Name:  "organization",
						Usage: "The organization ID to use for this project",
					},
				},
				Action: createProject,
			},
			{
				Name:   "archive",
				Usage:  "Archive an existing project",
				Action: archiveProject,
			},
			{
				Name:  "set",
				Usage: "Set the active project for syncing",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "all",
						Usage: "Include submodule projects in the selection",
					},
					&cli.StringFlag{
						Name:  "provider",
						Value: "claude.ai",
						Usage: "Specify the provider for repositories without .claudesync",
					},
				},
				Action: setProject,
			},
			{
				Name:  "ls",
				Usage: "List all projects in the active organization",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "all",
						Usage: "Include archived projects in the list",
					},
				},
				Action: listProjects,
			},
		},
	}
}

func createProject(c *cli.Context) error {
	cfg := config.GetConfig()
	providerName := c.String("provider")
	prov, err := provider.GetProvider(providerName, cfg)
	if err != nil {
		return err
	}

	organizationID := c.String("organization")
	if organizationID == "" {
		organizationID = cfg.Get("active_organization_id").(string)
	}

	name := c.String("name")
	if name == "" {
		name = filepath.Base(c.String("local-path"))
	}

	description := c.String("description")
	localPath := c.String("local-path")

	newProject, err := prov.CreateProject(organizationID, name, description)
	if err != nil {
		return err
	}

	fmt.Printf("Project '%s' (uuid: %s) has been created successfully.\n", newProject.Name, newProject.ID)

	cfg.Set("active_provider", providerName, true)
	cfg.Set("active_organization_id", organizationID, true)
	cfg.Set("active_project_id", newProject.ID, true)
	cfg.Set("active_project_name", newProject.Name, true)
	cfg.Set("local_path", localPath, true)

	claudesyncDir := filepath.Join(localPath, ".claudesync")
	err = os.MkdirAll(claudesyncDir, 0755)
	if err != nil {
		return err
	}

	fmt.Printf("\nProject setup complete. You can now start syncing files with this project. ")
	fmt.Printf("URL: https://claude.ai/project/%s\n", newProject.ID)

	return nil
}

func archiveProject(c *cli.Context) error {
	cfg := config.GetConfig()
	prov, err := provider.GetProvider(cfg.Get("active_provider").(string), cfg)
	if err != nil {
		return err
	}

	organizationID := cfg.Get("active_organization_id").(string)
	projects, err := prov.GetProjects(organizationID, false)
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No active projects found.")
		return nil
	}

	fmt.Println("Available projects to archive:")
	for i, proj := range projects {
		fmt.Printf("  %d. %s (ID: %s)\n", i+1, proj.Name, proj.ID)
	}

	var selection int
	fmt.Print("Enter the number of the project to archive: ")
	_, err = fmt.Scanf("%d", &selection)
	if err != nil || selection < 1 || selection > len(projects) {
		return fmt.Errorf("invalid selection")
	}

	selectedProject := projects[selection-1]
	fmt.Printf("Are you sure you want to archive the project '%s'? Archived projects cannot be modified but can still be viewed. (y/N): ", selectedProject.Name)
	var confirm string
	fmt.Scanf("%s", &confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("Archive operation cancelled.")
		return nil
	}

	err = prov.ArchiveProject(organizationID, selectedProject.ID)
	if err != nil {
		return err
	}

	fmt.Printf("Project '%s' has been archived.\n", selectedProject.Name)
	return nil
}

func setProject(c *cli.Context) error {
	cfg := config.GetConfig()
	providerName := c.String("provider")
	cfg.Set("active_provider", providerName, true)

	prov, err := provider.GetProvider(providerName, cfg)
	if err != nil {
		return err
	}

	organizationID := cfg.Get("active_organization_id").(string)
	showAll := c.Bool("all")

	projects, err := prov.GetProjects(organizationID, false)
	if err != nil {
		return err
	}

	var selectableProjects []providerapi.Project
	if showAll {
		selectableProjects = projects
	} else {
		for _, proj := range projects {
			if !isSubmoduleProject(proj.Name) {
				selectableProjects = append(selectableProjects, proj)
			}
		}
	}

	if len(selectableProjects) == 0 {
		fmt.Println("No active projects found.")
		return nil
	}

	fmt.Println("Available projects:")
	for i, proj := range selectableProjects {
		projectType := "Main Project"
		if isSubmoduleProject(proj.Name) {
			projectType = "Submodule"
		}
		fmt.Printf("  %d. %s (ID: %s) - %s\n", i+1, proj.Name, proj.ID, projectType)
	}

	var selection int
	fmt.Print("Enter the number of the project to select: ")
	_, err = fmt.Scanf("%d", &selection)
	if err != nil || selection < 1 || selection > len(selectableProjects) {
		return fmt.Errorf("invalid selection")
	}

	selectedProject := selectableProjects[selection-1]
	cfg.Set("active_project_id", selectedProject.ID, true)
	cfg.Set("active_project_name", selectedProject.Name, true)
	fmt.Printf("Selected project: %s (ID: %s)\n", selectedProject.Name, selectedProject.ID)

	// Create .claudesync directory in the current working directory if it doesn't exist
	err = os.MkdirAll(".claudesync", 0755)
	if err != nil {
		return err
	}
	fmt.Printf("Ensured .claudesync directory exists in %s\n", getCurrentWorkingDirectory())

	return nil
}

func listProjects(c *cli.Context) error {
	cfg := config.GetConfig()
	providerName := cfg.Get("active_provider")
	if providerName == nil {
		return fmt.Errorf("active provider not set. Please run 'claudesync auth login' first")
	}

	prov, err := provider.GetProvider(providerName.(string), cfg)
	if err != nil {
		return err
	}

	organizationID := cfg.Get("active_organization_id")
	if organizationID == nil {
		return fmt.Errorf("active organization not set. Please run 'claudesync organization set' first")
	}

	showAll := c.Bool("all")

	projects, err := prov.GetProjects(organizationID.(string), showAll)
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	fmt.Println("Remote projects:")
	for _, proj := range projects {
		status := ""
		if proj.ArchivedAt != nil {
			status = " (Archived)"
		}
		fmt.Printf("  - %s (ID: %s)%s\n", proj.Name, proj.ID, status)
	}

	return nil
}

func isSubmoduleProject(name string) bool {
	return false // Implement submodule detection logic
}

func getCurrentWorkingDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}
