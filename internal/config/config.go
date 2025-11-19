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
	Binance  BinanceConfig  `yaml:"binance"`
	Trading  TradingConfig  `yaml:"trading"`
	WebAPI   WebAPIConfig   `yaml:"webapi"`
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

// BinanceConfig contains Binance global settings (accounts are in database)
type BinanceConfig struct {
	BaseURL   string `yaml:"base_url"`    // REST API base URL (optional)
	WSBaseURL string `yaml:"ws_base_url"` // WebSocket base URL (optional)
}

// TradingConfig contains trading parameters
type TradingConfig struct {
	Enabled          bool    `yaml:"enabled"`
	Leverage         int     `yaml:"leverage"`
	OrderAmount      float64 `yaml:"order_amount"`       // Position size in USDT
	TargetPercent    float64 `yaml:"target_percent"`     // Take profit percentage (e.g., 0.02 for 2%)
	StopLossPercent  float64 `yaml:"stoploss_percent"`   // Stop loss percentage (e.g., 0.01 for 1%)
	OrderTimeout     int     `yaml:"order_timeout"`      // Timeout in seconds for TP/SL orders
	SignalPattern    string  `yaml:"signal_pattern"`     // Regex pattern for signal matching
	MaxPositions     int     `yaml:"max_positions"`      // Maximum concurrent positions
	DryRun           bool    `yaml:"dry_run"`            // If true, don't execute real orders
}

// WebAPIConfig contains web API server settings
type WebAPIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	CORSOrigins []string `yaml:"cors_origins"`
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

	// Validate trading config if trading is enabled
	if c.Trading.Enabled {
		if c.Trading.Leverage <= 0 || c.Trading.Leverage > 125 {
			return fmt.Errorf("trading.leverage must be between 1 and 125")
		}
		if c.Trading.OrderAmount <= 0 {
			return fmt.Errorf("trading.order_amount must be greater than 0")
		}
		if c.Trading.TargetPercent <= 0 {
			return fmt.Errorf("trading.target_percent must be greater than 0")
		}
		if c.Trading.StopLossPercent <= 0 {
			return fmt.Errorf("trading.stoploss_percent must be greater than 0")
		}
		if c.Trading.SignalPattern == "" {
			return fmt.Errorf("trading.signal_pattern is required when trading is enabled")
		}
	}

	return nil
}

// IsBot returns true if bot authentication is configured
func (c *Config) IsBot() bool {
	return c.Telegram.BotToken != ""
}
