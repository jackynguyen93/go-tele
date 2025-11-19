package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"tdlib-go/internal/cli"
	"tdlib-go/internal/config"
	"tdlib-go/internal/storage"
	"tdlib-go/internal/telegram"
	"tdlib-go/internal/trading"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "", "Log level (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := setupLogger(*logLevel)

	logger.Info("Starting Telegram Channel Monitor...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Override log level if specified in config
	if *logLevel == "" && cfg.Logging.Level != "" {
		setLogLevel(logger, cfg.Logging.Level)
	}

	// Create necessary directories
	if err := createDirectories(cfg); err != nil {
		logger.Fatalf("Failed to create directories: %v", err)
	}

	// Initialize database
	dbPath, err := storage.GetDatabasePath(cfg.Database.DSN)
	if err != nil {
		logger.Fatalf("Failed to get database path: %v", err)
	}

	repo, err := storage.NewRepository(dbPath)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer repo.Close()

	logger.Info("Database initialized successfully")

	// Initialize Telegram client
	client, err := telegram.NewClient(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create Telegram client: %v", err)
	}

	// Start client and authenticate
	if err := client.Start(); err != nil {
		logger.Fatalf("Failed to start Telegram client: %v", err)
	}

	// Initialize monitor
	monitor := telegram.NewMonitor(client, repo, cfg, logger)

	// Initialize trading engine
	tradingEngine, err := trading.NewEngine(repo, cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create trading engine: %v", err)
	}

	// Set message callback for trading
	monitor.SetMessageCallback(tradingEngine.ProcessMessage)

	// Start trading engine
	if err := tradingEngine.Start(); err != nil {
		logger.Fatalf("Failed to start trading engine: %v", err)
	}

	// Start monitoring
	if err := monitor.Start(); err != nil {
		logger.Fatalf("Failed to start monitor: %v", err)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start message listener in a goroutine
	listenerDone := make(chan error, 1)
	go func() {
		listenerDone <- client.StartListening()
	}()

	// Start CLI in a goroutine
	cliHandler := cli.NewCLI(monitor, logger)
	go func() {
		cliHandler.Start()
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		logger.Info("Shutdown signal received")
	case err := <-listenerDone:
		if err != nil {
			logger.Errorf("Listener error: %v", err)
		}
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	logger.Info("Initiating graceful shutdown...")

	// Stop trading engine
	if err := tradingEngine.Stop(); err != nil {
		logger.Errorf("Error stopping trading engine: %v", err)
	}

	// Stop client
	if err := client.Stop(); err != nil {
		logger.Errorf("Error stopping client: %v", err)
	}

	// Close database
	if err := repo.Close(); err != nil {
		logger.Errorf("Error closing database: %v", err)
	}

	logger.Info("Shutdown complete")
}

// setupLogger initializes and configures the logger
func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set log level
	if level != "" {
		setLogLevel(logger, level)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	return logger
}

// setLogLevel sets the log level
func setLogLevel(logger *logrus.Logger, level string) {
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
}

// createDirectories creates necessary directories for the application
func createDirectories(cfg *config.Config) error {
	dirs := []string{
		cfg.TDLib.DatabaseDirectory,
		cfg.TDLib.FilesDirectory,
		filepath.Dir(cfg.Database.DSN),
	}

	for _, dir := range dirs {
		if dir == "" || dir == "." {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// reconnect attempts to reconnect to Telegram
func reconnect(client *telegram.Client, logger *logrus.Logger, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		logger.Infof("Reconnection attempt %d/%d", i+1, maxRetries)

		if err := client.Start(); err != nil {
			logger.Errorf("Reconnection failed: %v", err)
			time.Sleep(time.Duration(i+1) * 5 * time.Second)
			continue
		}

		logger.Info("Reconnection successful")
		return nil
	}

	return fmt.Errorf("failed to reconnect after %d attempts", maxRetries)
}
