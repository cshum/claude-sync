package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type Config struct {
	GlobalConfig map[string]interface{}
	LocalConfig  map[string]interface{}
	configDir    string
}

var instance *Config

func GetConfig() *Config {
	if instance == nil {
		instance = &Config{
			GlobalConfig: make(map[string]interface{}),
			LocalConfig:  make(map[string]interface{}),
			configDir:    filepath.Join(os.Getenv("HOME"), ".claudesync"),
		}
		instance.loadGlobalConfig()
		instance.loadLocalConfig()
	}
	return instance
}

func (c *Config) loadGlobalConfig() {
	configFile := filepath.Join(c.configDir, "config.json")
	data, err := os.ReadFile(configFile)
	if err == nil {
		json.Unmarshal(data, &c.GlobalConfig)
	}
}

func (c *Config) loadLocalConfig() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		configFile := filepath.Join(cwd, ".claudesync", "config.local.json")
		data, err := os.ReadFile(configFile)
		if err == nil {
			json.Unmarshal(data, &c.LocalConfig)
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
}

func (c *Config) Get(key string) interface{} {
	if value, ok := c.LocalConfig[key]; ok {
		return value
	}
	return c.GlobalConfig[key]
}

func (c *Config) Set(key string, value interface{}, local bool) error {
	if local {
		c.LocalConfig[key] = value
		return c.saveLocalConfig()
	}
	c.GlobalConfig[key] = value
	return c.saveGlobalConfig()
}

func (c *Config) saveGlobalConfig() error {
	data, err := json.MarshalIndent(c.GlobalConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.configDir, "config.json"), data, 0644)
}

func (c *Config) saveLocalConfig() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	configDir := filepath.Join(cwd, ".claudesync")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c.LocalConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "config.local.json"), data, 0644)
}

func (c *Config) SetSessionKey(provider, sessionKey string, expiry time.Time) error {
	c.GlobalConfig[fmt.Sprintf("%s_session_key", provider)] = map[string]string{
		"key":    sessionKey,
		"expiry": expiry.Format(time.RFC3339),
	}
	return c.saveGlobalConfig()
}

func (c *Config) GetSessionKey(provider string) (string, time.Time, error) {
	key := fmt.Sprintf("%s_session_key", provider)
	value, ok := c.GlobalConfig[key]
	if !ok {
		return "", time.Time{}, fmt.Errorf("no session key found for provider %s", provider)
	}

	sessionKeyMap, ok := value.(map[string]interface{})
	if !ok {
		return "", time.Time{}, fmt.Errorf("invalid session key format for provider %s", provider)
	}

	sessionKey, ok := sessionKeyMap["key"].(string)
	if !ok {
		return "", time.Time{}, fmt.Errorf("invalid session key format for provider %s", provider)
	}

	expiryStr, ok := sessionKeyMap["expiry"].(string)
	if !ok {
		return "", time.Time{}, fmt.Errorf("invalid session key format for provider %s", provider)
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return "", time.Time{}, err
	}

	return sessionKey, expiry, nil
}

func (c *Config) ClearAllSessionKeys() {
	for key := range c.GlobalConfig {
		if strings.HasSuffix(key, "_session_key") {
			delete(c.GlobalConfig, key)
		}
	}
	c.saveGlobalConfig()
}

func (c *Config) GetProvidersWithSessionKeys() []string {
	var providers []string
	for key := range c.GlobalConfig {
		if strings.HasSuffix(key, "_session_key") {
			providers = append(providers, strings.TrimSuffix(key, "_session_key"))
		}
	}
	return providers
}

func Command() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage claudesync configuration",
		Subcommands: []*cli.Command{
			{
				Name:      "set",
				Usage:     "Set a configuration value",
				ArgsUsage: "<key> <value>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "local",
						Usage: "Set the value in the local configuration",
					},
				},
				Action: setConfig,
			},
			{
				Name:      "get",
				Usage:     "Get a configuration value",
				ArgsUsage: "<key>",
				Action:    getConfig,
			},
			{
				Name:   "ls",
				Usage:  "List all configuration values",
				Action: listConfig,
			},
		},
	}
}

func setConfig(c *cli.Context) error {
	if c.NArg() != 2 {
		return fmt.Errorf("please provide both key and value")
	}

	key := c.Args().Get(0)
	value := c.Args().Get(1)
	local := c.Bool("local")

	cfg := GetConfig()
	return cfg.Set(key, value, local)
}

func getConfig(c *cli.Context) error {
	if c.NArg() != 1 {
		return fmt.Errorf("please provide a key")
	}

	key := c.Args().Get(0)
	cfg := GetConfig()
	value := cfg.Get(key)

	if value == nil {
		fmt.Printf("Configuration %s is not set\n", key)
	} else {
		fmt.Printf("%s: %v\n", key, value)
	}
	return nil
}

func listConfig(c *cli.Context) error {
	cfg := GetConfig()
	combinedConfig := make(map[string]interface{})

	for k, v := range cfg.GlobalConfig {
		combinedConfig[k] = v
	}
	for k, v := range cfg.LocalConfig {
		combinedConfig[k] = v
	}

	data, err := json.MarshalIndent(combinedConfig, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
