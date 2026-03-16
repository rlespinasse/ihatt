package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ScanRoots []string `mapstructure:"scan_roots"`
}

func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "ihatt")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ihatt")
}

func DataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "ihatt")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "ihatt")
}

func DBPath() string {
	return filepath.Join(DataDir(), "ihatt.db")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	v.SetDefault("scan_roots", []string{})

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	v.Set("scan_roots", cfg.ScanRoots)

	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	return v.WriteConfigAs(ConfigPath())
}

func EnsureDirs() error {
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	return os.MkdirAll(DataDir(), 0o755)
}
