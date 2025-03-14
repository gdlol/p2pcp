package config

// spell-checker: ignore adrg

import (
	"encoding/json"
	"p2pcp/internal/errors"
	"path/filepath"
	"project/pkg/project"
	"strings"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

type Config struct {
	BootstrapPeers []string
}

func NewConfig() Config {
	return Config{
		BootstrapPeers: nil,
	}
}

func initializeConfig() {
	xdg.Reload()
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(filepath.Join(xdg.ConfigHome, project.Name))
	defaultConfig := NewConfig()
	jsonString, err := json.Marshal(defaultConfig)
	errors.Unexpected(err, "initializeConfig: Marshal default config")
	viper.ReadConfig(strings.NewReader(string(jsonString)))
}

func init() {
	initializeConfig()
}

var config = NewConfig()

func LoadConfig() error {
	if err := viper.MergeInConfig(); err != nil {
		return nil
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err == nil {
		config = cfg
	}
	return nil
}

func GetConfig() Config {
	return config
}
