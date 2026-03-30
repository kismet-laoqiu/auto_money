package catalog

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DSN                   string        `yaml:"dsn"`
	MaxOpenConns          int           `yaml:"max_open_conns"`
	MaxIdleConns          int           `yaml:"max_idle_conns"`
	RetentionDays         int           `yaml:"retention_days"`
	CompressionAfterHours int           `yaml:"compression_after_hours"`
	HealthTimeout         time.Duration `yaml:"health_timeout"`
}

func LoadConfig(path string) (Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read warehouse config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode warehouse config %s: %w", path, err)
	}
	applyDefaults(&cfg)
	if cfg.DSN == "" {
		return Config{}, fmt.Errorf("warehouse dsn is required")
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 8
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 4
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 365
	}
	if cfg.CompressionAfterHours <= 0 {
		cfg.CompressionAfterHours = 24
	}
	if cfg.HealthTimeout <= 0 {
		cfg.HealthTimeout = 5 * time.Second
	}
}
