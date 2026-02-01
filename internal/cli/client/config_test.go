package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	require.NoError(t, err)
	assert.NotEmpty(t, dir)
	assert.True(t, filepath.IsAbs(dir))
	assert.True(t, strings.HasSuffix(dir, "neotex"))
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.True(t, filepath.IsAbs(path))
	assert.True(t, strings.HasSuffix(path, "config.json"))
}

func TestLoadGlobalConfig_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	config, err := LoadGlobalConfig()
	require.NoError(t, err)
	assert.Nil(t, config)
}

func TestLoadGlobalConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := GlobalConfig{
		APIKey: "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		APIURL: "http://localhost:8080",
	}
	data, _ := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0600))

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	config, err := LoadGlobalConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, testConfig.APIKey, config.APIKey)
	assert.Equal(t, testConfig.APIURL, config.APIURL)
}

func TestLoadGlobalConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	require.NoError(t, os.WriteFile(configPath, []byte("{invalid json}"), 0600))

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	config, err := LoadGlobalConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestSaveGlobalConfig_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "neotex")
	configPath := filepath.Join(configDir, "config.json")

	oldGetConfigDir := getConfigDirFunc
	oldGetConfigPath := getConfigPathFunc
	getConfigDirFunc = func() (string, error) {
		return configDir, nil
	}
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		getConfigDirFunc = oldGetConfigDir
		getConfigPathFunc = oldGetConfigPath
	}()

	config := &GlobalConfig{
		APIKey: "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		APIURL: "http://localhost:8080",
	}

	err := SaveGlobalConfig(config)
	require.NoError(t, err)

	assert.DirExists(t, configDir)
	assert.FileExists(t, configPath)
}

func TestSaveGlobalConfig_SetCorrectPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	oldGetConfigDir := getConfigDirFunc
	oldGetConfigPath := getConfigPathFunc
	getConfigDirFunc = func() (string, error) {
		return tmpDir, nil
	}
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		getConfigDirFunc = oldGetConfigDir
		getConfigPathFunc = oldGetConfigPath
	}()

	config := &GlobalConfig{
		APIKey: "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		APIURL: "http://localhost:8080",
	}

	err := SaveGlobalConfig(config)
	require.NoError(t, err)

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSaveGlobalConfig_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	oldGetConfigDir := getConfigDirFunc
	oldGetConfigPath := getConfigPathFunc
	getConfigDirFunc = func() (string, error) {
		return tmpDir, nil
	}
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		getConfigDirFunc = oldGetConfigDir
		getConfigPathFunc = oldGetConfigPath
	}()

	config := &GlobalConfig{
		APIKey: "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		APIURL: "http://localhost:8080",
	}

	err := SaveGlobalConfig(config)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var loaded GlobalConfig
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.Equal(t, config.APIKey, loaded.APIKey)
	assert.Equal(t, config.APIURL, loaded.APIURL)
}

func TestSaveGlobalConfig_NilConfig(t *testing.T) {
	err := SaveGlobalConfig(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestDeleteGlobalConfig_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0600))

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	err := DeleteGlobalConfig()
	require.NoError(t, err)
	assert.NoFileExists(t, configPath)
}

func TestDeleteGlobalConfig_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	err := DeleteGlobalConfig()
	require.NoError(t, err)
}

func TestIsValidAPIKey_ValidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"valid lowercase", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", true},
		{"valid uppercase", "ntx_0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF", true},
		{"valid mixed case", "ntx_0123456789AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789AbCdEf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidAPIKey(tt.key))
		})
	}
}

func TestIsValidAPIKey_InvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"missing prefix", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"wrong prefix", "abc_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"too short", "ntx_0123456789abcdef", false},
		{"too long", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00", false},
		{"invalid chars", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg", false},
		{"invalid chars space", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde ", false},
		{"empty", "", false},
		{"only prefix", "ntx_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidAPIKey(tt.key))
		})
	}
}

func TestGetCredentialSource_FlagPriority(t *testing.T) {
	t.Setenv("NEOTEX_API_KEY", "")
	t.Setenv("NEOTEX_API_URL", "")

	source, key, url := GetCredentialSource(
		"ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"http://localhost:8080",
	)

	assert.Equal(t, SourceFlag, source)
	assert.Equal(t, "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", key)
	assert.Equal(t, "http://localhost:8080", url)
}

func TestGetCredentialSource_EnvFilePriority(t *testing.T) {
	t.Setenv("NEOTEX_API_KEY", "ntx_envkey0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	t.Setenv("NEOTEX_API_URL", "http://env:8080")

	source, key, url := GetCredentialSource("", "")

	assert.Equal(t, SourceEnvFile, source)
	assert.Equal(t, "ntx_envkey0123456789abcdef0123456789abcdef0123456789abcdef0123456789", key)
	assert.Equal(t, "http://env:8080", url)
}

func TestGetCredentialSource_GlobalConfigPriority(t *testing.T) {
	t.Setenv("NEOTEX_API_KEY", "")
	t.Setenv("NEOTEX_API_URL", "")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := GlobalConfig{
		APIKey: "ntx_globalkey123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		APIURL: "http://global:8080",
	}
	data, _ := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0600))

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	source, key, url := GetCredentialSource("", "")

	assert.Equal(t, SourceGlobalConfig, source)
	assert.Equal(t, testConfig.APIKey, key)
	assert.Equal(t, testConfig.APIURL, url)
}

func TestGetCredentialSource_NoCredentials(t *testing.T) {
	t.Setenv("NEOTEX_API_KEY", "")
	t.Setenv("NEOTEX_API_URL", "")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	source, key, url := GetCredentialSource("", "")

	assert.Equal(t, SourceNone, source)
	assert.Empty(t, key)
	assert.Empty(t, url)
}

func TestGetCredentialSource_PartialEnvVars(t *testing.T) {
	t.Setenv("NEOTEX_API_KEY", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("NEOTEX_API_URL", "")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	source, key, url := GetCredentialSource("", "")

	assert.Equal(t, SourceNone, source)
	assert.Empty(t, key)
	assert.Empty(t, url)
}

func TestGetCredentialSource_FlagOverridesEnv(t *testing.T) {
	t.Setenv("NEOTEX_API_KEY", "ntx_envkey0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	t.Setenv("NEOTEX_API_URL", "http://env:8080")

	source, key, url := GetCredentialSource(
		"ntx_flagkey123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		"http://flag:8080",
	)

	assert.Equal(t, SourceFlag, source)
	assert.Equal(t, "ntx_flagkey123456789abcdef0123456789abcdef0123456789abcdef0123456789", key)
	assert.Equal(t, "http://flag:8080", url)
}

func TestGetCredentialSource_EnvOverridesGlobalConfig(t *testing.T) {
	t.Setenv("NEOTEX_API_KEY", "ntx_envkey0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	t.Setenv("NEOTEX_API_URL", "http://env:8080")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := GlobalConfig{
		APIKey: "ntx_globalkey123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		APIURL: "http://global:8080",
	}
	data, _ := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0600))

	oldGetConfigPath := getConfigPathFunc
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { getConfigPathFunc = oldGetConfigPath }()

	source, key, url := GetCredentialSource("", "")

	assert.Equal(t, SourceEnvFile, source)
	assert.Equal(t, "ntx_envkey0123456789abcdef0123456789abcdef0123456789abcdef0123456789", key)
	assert.Equal(t, "http://env:8080", url)
}

func TestRoundTrip_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	oldGetConfigDir := getConfigDirFunc
	oldGetConfigPath := getConfigPathFunc
	getConfigDirFunc = func() (string, error) {
		return tmpDir, nil
	}
	getConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		getConfigDirFunc = oldGetConfigDir
		getConfigPathFunc = oldGetConfigPath
	}()

	originalConfig := &GlobalConfig{
		APIKey: "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		APIURL: "http://localhost:8080",
	}
	err := SaveGlobalConfig(originalConfig)
	require.NoError(t, err)

	loadedConfig, err := LoadGlobalConfig()
	require.NoError(t, err)
	require.NotNil(t, loadedConfig)

	assert.Equal(t, originalConfig.APIKey, loadedConfig.APIKey)
	assert.Equal(t, originalConfig.APIURL, loadedConfig.APIURL)
}
