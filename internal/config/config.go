package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Database DatabaseConfig `yaml:"database"`
	TDLib    TDLibConfig    `yaml:"tdlib"`
	Channels []string       `yaml:"channels"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// TelegramConfig contains Telegram API credentials
type TelegramConfig struct {
	APIID       int32  `yaml:"api_id"`
	APIHash     string `yaml:"api_hash"`
	PhoneNumber string `yaml:"phone_number,omitempty"`
	BotToken    string `yaml:"bot_token,omitempty"`
	UseTestDC   bool   `yaml:"use_test_dc"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Type string `yaml:"type"` // sqlite, postgres, mysql
	DSN  string `yaml:"dsn"`  // Data Source Name
}

// TDLibConfig contains TDLib parameters
type TDLibConfig struct {
	DatabaseDirectory string `yaml:"database_directory"`
	FilesDirectory    string `yaml:"files_directory"`
	UseFileDatabase   bool   `yaml:"use_file_database"`
	UseChatInfoDB     bool   `yaml:"use_chat_info_database"`
	UseMessageDB      bool   `yaml:"use_message_database"`
	SystemLanguage    string `yaml:"system_language"`
	DeviceModel       string `yaml:"device_model"`
	SystemVersion     string `yaml:"system_version"`
	AppVersion        string `yaml:"app_version"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // text, json
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Telegram.APIID == 0 {
		return fmt.Errorf("telegram.api_id is required")
	}
	if c.Telegram.APIHash == "" {
		return fmt.Errorf("telegram.api_hash is required")
	}
	if c.Telegram.PhoneNumber == "" && c.Telegram.BotToken == "" {
		return fmt.Errorf("either telegram.phone_number or telegram.bot_token must be provided")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if c.TDLib.DatabaseDirectory == "" {
		return fmt.Errorf("tdlib.database_directory is required")
	}
	if c.TDLib.FilesDirectory == "" {
		return fmt.Errorf("tdlib.files_directory is required")
	}
	return nil
}

// IsBot returns true if bot authentication is configured
func (c *Config) IsBot() bool {
	return c.Telegram.BotToken != ""
}
