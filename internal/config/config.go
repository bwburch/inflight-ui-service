package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Logging    LoggingConfig    `yaml:"logging"`
	Migrations MigrationsConfig `yaml:"migrations"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type DatabaseConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	User               string `yaml:"user"`
	Password           string `yaml:"password"`
	DBName             string `yaml:"dbname"`
	SSLMode            string `yaml:"sslmode"`
	MaxConnections     int    `yaml:"max_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type MigrationsConfig struct {
	AutoRun bool   `yaml:"auto_run"`
	Path    string `yaml:"path"`
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply environment variable overrides
	if val := os.Getenv("SERVER_PORT"); val != "" {
		cfg.Server.Port = val
	}
	if val := os.Getenv("DB_HOST"); val != "" {
		cfg.Database.Host = val
	}
	if val := os.Getenv("DB_USER"); val != "" {
		cfg.Database.User = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		cfg.Database.Password = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		cfg.Database.DBName = val
	}

	return &cfg, nil
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}
