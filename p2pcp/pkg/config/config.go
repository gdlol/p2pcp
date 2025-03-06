package config

// spell-checker: ignore adrg

import (
	"p2pcp/internal/transfer"
	"path/filepath"
	"project/pkg/project"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

type Config struct {
	BootstrapPeers []string
	PayloadSize    uint16
}

func NewConfig() Config {
	return Config{
		BootstrapPeers: nil,
		PayloadSize:    transfer.DefaultPayloadSize,
	}
}

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(filepath.Join(xdg.ConfigHome, project.Name))
	defaultConfig := NewConfig()
	viper.SetDefault("BootstrapPeers", defaultConfig.BootstrapPeers)
	viper.SetDefault("PayloadSize", defaultConfig.PayloadSize)
}

var config = NewConfig()

func LoadConfig() error {
	if err := viper.ReadInConfig(); err != nil {
		return nil
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}
	config = cfg
	return nil
}

func GetConfig() Config {
	return config
}
