package sync

import (
	"fmt"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/internal/provider"
	"github.com/cshum/claude-sync/v2/pkg/utils"
	"github.com/cshum/claude-sync/v2/providerapi"
	"github.com/urfave/cli/v2"
	"io/ioutil"
)

type SyncManager struct {
	provider providerapi.Provider
	config   *config.Config
}

func NewSyncManager(p providerapi.Provider, c *config.Config) *SyncManager {
	return &SyncManager{
		provider: p,
		config:   c,
	}
}

func (sm *SyncManager) Sync(localFiles map[string]string, remoteFiles []providerapi.File) error {
	organizationID := sm.config.Get("active_organization_id").(string)
	projectID := sm.config.Get("active_project_id").(string)

	// Files to upload or update
	for localPath, localHash := range localFiles {
		found := false
		for _, remoteFile := range remoteFiles {
			if remoteFile.FileName == localPath {
				found = true
				if localHash != remoteFile.Content {
					// Update file
					content, err := ioutil.ReadFile(localPath)
					if err != nil {
						return fmt.Errorf("failed to read local file %s: %v", localPath, err)
					}
					err = sm.provider.UploadFile(organizationID, projectID, localPath, string(content))
					if err != nil {
						return fmt.Errorf("failed to update file %s: %v", localPath, err)
					}
					fmt.Printf("Updated file: %s\n", localPath)
				}
				break
			}
		}
		if !found {
			// Upload new file
			content, err := ioutil.ReadFile(localPath)
			if err != nil {
				return fmt.Errorf("failed to read local file %s: %v", localPath, err)
			}
			err = sm.provider.UploadFile(organizationID, projectID, localPath, string(content))
			if err != nil {
				return fmt.Errorf("failed to upload file %s: %v", localPath, err)
			}
			fmt.Printf("Uploaded new file: %s\n", localPath)
		}
	}

	// Files to delete
	for _, remoteFile := range remoteFiles {
		if _, exists := localFiles[remoteFile.FileName]; !exists {
			err := sm.provider.DeleteFile(organizationID, projectID, remoteFile.UUID)
			if err != nil {
				return fmt.Errorf("failed to delete file %s: %v", remoteFile.FileName, err)
			}
			fmt.Printf("Deleted file: %s\n", remoteFile.FileName)
		}
	}

	return nil
}

func Command() *cli.Command {
	return &cli.Command{
		Name:  "push",
		Usage: "Synchronize the project files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "category",
				Usage: "Specify the file category to sync",
			},
			&cli.BoolFlag{
				Name:  "uberproject",
				Usage: "Include submodules in the parent project sync",
			},
		},
		Action: pushAction,
	}
}

func pushAction(c *cli.Context) error {
	cfg := config.GetConfig()
	prov, err := provider.GetProvider(cfg.Get("active_provider").(string), cfg)
	if err != nil {
		return err
	}

	organizationID := cfg.Get("active_organization_id").(string)
	projectID := cfg.Get("active_project_id").(string)
	localPath := cfg.Get("local_path").(string)

	sm := NewSyncManager(prov, cfg)

	localFiles, err := utils.GetLocalFiles(localPath)
	if err != nil {
		return fmt.Errorf("failed to get local files: %v", err)
	}

	remoteFiles, err := prov.ListFiles(organizationID, projectID)
	if err != nil {
		return fmt.Errorf("failed to list remote files: %v", err)
	}

	err = sm.Sync(localFiles, remoteFiles)
	if err != nil {
		return fmt.Errorf("sync failed: %v", err)
	}

	fmt.Println("Sync completed successfully")
	return nil
}
