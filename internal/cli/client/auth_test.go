package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthLogin_StoresCredentials(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	apiKey := "ntx_" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	apiURL := "http://localhost:8080"

	err := runAuthLogin(apiKey, apiURL)
	require.NoError(t, err)

	config, err := LoadGlobalConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, apiKey, config.APIKey)
	assert.Equal(t, apiURL, config.APIURL)
}

func TestAuthLogin_OverwritesExisting(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	oldKey := "ntx_" + "0000000000000000000000000000000000000000000000000000000000000000"
	oldURL := "http://old.example.com"
	err := SaveGlobalConfig(&GlobalConfig{APIKey: oldKey, APIURL: oldURL})
	require.NoError(t, err)

	newKey := "ntx_" + "1111111111111111111111111111111111111111111111111111111111111111"
	newURL := "http://new.example.com"
	err = runAuthLogin(newKey, newURL)
	require.NoError(t, err)

	config, err := LoadGlobalConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, newKey, config.APIKey)
	assert.Equal(t, newURL, config.APIURL)
}

func TestAuthLogin_ValidatesKeyFormat(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	invalidKey := "invalid_key"
	err := runAuthLogin(invalidKey, "http://localhost:8080")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key format")

	config, err := LoadGlobalConfig()
	require.NoError(t, err)
	assert.Nil(t, config)
}

func TestAuthLogout_ClearsGlobalConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	apiKey := "ntx_" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	err := SaveGlobalConfig(&GlobalConfig{APIKey: apiKey, APIURL: "http://localhost:8080"})
	require.NoError(t, err)

	err = runAuthLogout()
	require.NoError(t, err)

	config, err := LoadGlobalConfig()
	require.NoError(t, err)
	assert.Nil(t, config)
}

func TestAuthLogout_IdempotentWhenNoConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	err := runAuthLogout()
	require.NoError(t, err)

	err = runAuthLogout()
	require.NoError(t, err)
}

func TestAuthStatus_ShowsGlobalSource(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	apiKey := "ntx_" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	apiURL := "http://localhost:8080"
	err := SaveGlobalConfig(&GlobalConfig{APIKey: apiKey, APIURL: apiURL})
	require.NoError(t, err)

	err = runAuthStatus(false)
	require.NoError(t, err)
}

func TestAuthStatus_ShowsEnvSource(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	apiKey := "ntx_" + "e1e2e3e4e5e6e1e2e3e4e5e6e1e2e3e4e5e6e1e2e3e4e5e6e1e2e3e4e5e6e1e2"
	apiURL := "http://env.example.com"
	t.Setenv("NEOTEX_API_KEY", apiKey)
	t.Setenv("NEOTEX_API_URL", apiURL)

	err := runAuthStatus(false)
	require.NoError(t, err)
}

func TestAuthStatus_ShowsNoAuth(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	err := runAuthStatus(false)
	require.NoError(t, err)
}

func TestAuthStatus_JSONOutput(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalGetConfigDir := getConfigDirFunc
	originalGetConfigPath := getConfigPathFunc
	defer func() {
		getConfigDirFunc = originalGetConfigDir
		getConfigPathFunc = originalGetConfigPath
	}()

	getConfigDirFunc = func() (string, error) { return tempDir, nil }
	getConfigPathFunc = func() (string, error) { return configPath, nil }

	apiKey := "ntx_" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	apiURL := "http://localhost:8080"
	err := SaveGlobalConfig(&GlobalConfig{APIKey: apiKey, APIURL: apiURL})
	require.NoError(t, err)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runAuthStatus(true)
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Equal(t, true, result["authenticated"])
	assert.Equal(t, "global_config", result["source"])
	assert.Equal(t, "ntx_a1b...a1b2", result["api_key"])
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid key",
			input:    "ntx_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			expected: "ntx_a1b...a1b2",
		},
		{
			name:     "short key",
			input:    "short",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
