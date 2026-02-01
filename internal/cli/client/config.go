package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GlobalConfig represents the global authentication configuration stored in config.json
type GlobalConfig struct {
	APIKey string `json:"api_key"`
	APIURL string `json:"api_url"`
}

var (
	getConfigDirFunc  = defaultGetConfigDir
	getConfigPathFunc = defaultGetConfigPath
)

func defaultGetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "neotex"), nil
}

func defaultGetConfigPath() (string, error) {
	configDir, err := getConfigDirFunc()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// GetConfigDir returns the platform-specific configuration directory
func GetConfigDir() (string, error) {
	return getConfigDirFunc()
}

// GetConfigPath returns the full path to the config.json file
func GetConfigPath() (string, error) {
	return getConfigPathFunc()
}

// LoadGlobalConfig reads and parses the global config.json file
// Returns nil config (not error) if file doesn't exist
func LoadGlobalConfig() (*GlobalConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveGlobalConfig writes the config to config.json with 0600 permissions
func SaveGlobalConfig(config *GlobalConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DeleteGlobalConfig removes the config.json file
func DeleteGlobalConfig() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	if err := os.Remove(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete config file: %w", err)
	}

	return nil
}

// IsValidAPIKey validates the API key format: ntx_ + 64 hex chars
func IsValidAPIKey(key string) bool {
	if !strings.HasPrefix(key, "ntx_") {
		return false
	}
	hexPart := key[4:] // Skip "ntx_" prefix
	if len(hexPart) != 64 {
		return false
	}
	// Check if all characters are valid hex
	matched, _ := regexp.MatchString("^[0-9a-fA-F]{64}$", hexPart)
	return matched
}

// CredentialSource represents where credentials came from
type CredentialSource string

const (
	SourceFlag         CredentialSource = "flag"
	SourceEnvFile      CredentialSource = "env_file"
	SourceGlobalConfig CredentialSource = "global_config"
	SourceNone         CredentialSource = "none"
)

// GetCredentialSource returns the source of credentials with cascade check
// Checks in order: flag -> env_file -> global_config -> none
func GetCredentialSource(flagAPIKey, flagAPIURL string) (CredentialSource, string, string) {
	// Check flags first
	if flagAPIKey != "" && flagAPIURL != "" {
		return SourceFlag, flagAPIKey, flagAPIURL
	}

	// Check environment variables
	envAPIKey := os.Getenv("NEOTEX_API_KEY")
	envAPIURL := os.Getenv("NEOTEX_API_URL")
	if envAPIKey != "" && envAPIURL != "" {
		return SourceEnvFile, envAPIKey, envAPIURL
	}

	// Check global config
	config, err := LoadGlobalConfig()
	if err == nil && config != nil && config.APIKey != "" && config.APIURL != "" {
		return SourceGlobalConfig, config.APIKey, config.APIURL
	}

	// No credentials found
	return SourceNone, "", ""
}
