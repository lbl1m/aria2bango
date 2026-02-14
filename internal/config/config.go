// Package config provides configuration management for aria2bango
package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Aria2     Aria2Config     `yaml:"aria2"`
	Detection DetectionConfig `yaml:"detection"`
	Blocking  BlockingConfig  `yaml:"blocking"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// Aria2Config holds aria2 RPC connection settings
type Aria2Config struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Secret       string        `yaml:"secret"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

// DetectionConfig holds detection rule settings
type DetectionConfig struct {
	Behavior BehaviorConfig `yaml:"behavior"`
}

// BehaviorConfig holds behavior analysis settings
type BehaviorConfig struct {
	Enabled          bool    `yaml:"enabled"`
	MinShareRatio    float64 `yaml:"min_share_ratio"`
	MinDataThreshold int64   `yaml:"min_data_threshold"`
}

// BlockingConfig holds blocking settings
type BlockingConfig struct {
	BaseDuration time.Duration `yaml:"base_duration"` // 基础屏蔽时长，累加惩罚的基数
	NftTable     string        `yaml:"nft_table"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Aria2: Aria2Config{
			Host:         "127.0.0.1",
			Port:         6800,
			Secret:       "",
			PollInterval: 10 * time.Second,
		},
		Detection: DetectionConfig{
			Behavior: BehaviorConfig{
				Enabled:          true,
				MinShareRatio:    0.1,
				MinDataThreshold: 10 * 1024 * 1024, // 10MB
			},
		},
		Blocking: BlockingConfig{
			BaseDuration: 5 * time.Minute, // 基础屏蔽5分钟，累加惩罚
			NftTable:     "aria2bango",
		},
		Logging: LoggingConfig{
			Level:      "info",
			File:       "/var/log/aria2bango/blocked.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
		},
	}
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// Save saves the configuration to a YAML file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
