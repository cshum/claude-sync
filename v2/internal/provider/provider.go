package provider

import (
	"fmt"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/internal/provider/claudeai"
	"github.com/cshum/claude-sync/v2/providerapi"
)

func GetProvider(providerName string, cfg *config.Config) (providerapi.Provider, error) {
	switch providerName {
	case "claude.ai":
		return claudeai.NewClaudeAIProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}
