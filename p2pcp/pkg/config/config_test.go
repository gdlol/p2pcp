package config

import (
	"os"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	configPath := filepath.Join(os.TempDir(), "p2pcp/test/config")
	appConfigPath := filepath.Join(configPath, project.Name)

	config1 := "{ \"BootstrapPeers\": [\"peer1\", \"peer2\"] }"
	func() {
		workspace.ResetDir(appConfigPath)
		viper.Reset()
		restore := workspace.SetEnv("XDG_CONFIG_HOME", configPath)
		defer restore()

		file, err := os.Create(filepath.Join(appConfigPath, "config.json"))
		require.NoError(t, err)
		file.WriteString(config1)
		file.Close()
		require.NoError(t, err)

		initializeConfig()
		err = LoadConfig()
		require.NoError(t, err)

		config := GetConfig()
		require.Equal(t, []string{"peer1", "peer2"}, config.BootstrapPeers)
	}()

	// Lowercase
	config2 := "{ \"bootstrapPeers\": [\"peer1\", \"peer2\"] }"
	func() {
		workspace.ResetDir(appConfigPath)
		viper.Reset()
		restore := workspace.SetEnv("XDG_CONFIG_HOME", configPath)
		defer restore()

		file, err := os.Create(filepath.Join(appConfigPath, "config.json"))
		require.NoError(t, err)
		file.WriteString(config2)
		file.Close()
		require.NoError(t, err)

		initializeConfig()
		err = LoadConfig()
		require.NoError(t, err)

		config := GetConfig()
		require.Equal(t, []string{"peer1", "peer2"}, config.BootstrapPeers)
	}()

	// Invalid
	config3 := "{ \"abc\": 1 }"
	func() {
		workspace.ResetDir(appConfigPath)
		viper.Reset()
		restore := workspace.SetEnv("XDG_CONFIG_HOME", configPath)
		defer restore()

		file, err := os.Create(filepath.Join(appConfigPath, "config.json"))
		require.NoError(t, err)
		file.WriteString(config3)
		file.Close()
		require.NoError(t, err)

		initializeConfig()
		err = LoadConfig()
		require.NoError(t, err)
		config := GetConfig()
		assert.Empty(t, config.BootstrapPeers)
	}()
}
