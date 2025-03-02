package config

import (
	"p2pcp/internal/transfer"

	"github.com/spf13/viper"
)

type Config struct {
	BootstrapPeers []string
	PayloadSize    uint16
}

var config = Config{
	BootstrapPeers: nil,
	PayloadSize:    transfer.DefaultPayloadSize,
}

func LoadConfig() error {
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
