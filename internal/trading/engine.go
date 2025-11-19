package trading

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"tdlib-go/internal/binance"
	"tdlib-go/internal/config"
	"tdlib-go/internal/storage"
	"tdlib-go/internal/webapi"
	"tdlib-go/pkg/models"
)

// Engine is the main trading engine
type Engine struct {
	parser   *SignalParser
	executor *OrderExecutor
	binance  *binance.Client
	repo     *storage.Repository
	webapi   *webapi.Server
	config   *config.Config
	logger   *logrus.Logger
}

// NewEngine creates a new trading engine
func NewEngine(repo *storage.Repository, cfg *config.Config, logger *logrus.Logger) (*Engine, error) {
	if !cfg.Trading.Enabled {
		logger.Info("Trading is disabled in configuration")
		return &Engine{
			repo:           repo,
			config:         cfg,
			logger:         logger,
			binanceClients: make(map[int64]*binance.Client),
		}, nil
	}

	// Initialize signal parser
	parser, err := NewSignalParser(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create signal parser: %w", err)
	}

	// Load Binance accounts from database
	accounts, err := repo.GetActiveAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to load Binance accounts: %w", err)
	}

	if len(accounts) == 0 {
		logger.Warn("No active Binance accounts found in database. Trading is enabled but no accounts configured.")
		logger.Warn("Please add Binance accounts via the web dashboard at /api/accounts")
	}

	// Create Binance clients for each account
	binanceClients := make(map[int64]*binance.Client)
	for _, account := range accounts {
		client := binance.NewClient(account.APIKey, account.APISecret, account.IsTestnet, logger)
		binanceClients[account.ID] = client
		logger.Infof("Initialized Binance client for account: %s (ID: %d, Testnet: %v)",
			account.Name, account.ID, account.IsTestnet)
	}

	// Get default account for executor
	defaultAccount, err := repo.GetDefaultAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to get default account: %w", err)
	}

	var executor *OrderExecutor
	if defaultAccount != nil && binanceClients[defaultAccount.ID] != nil {
		executor = NewOrderExecutor(binanceClients[defaultAccount.ID], repo, cfg, logger)
		executor.accountID = defaultAccount.ID // Set account ID in executor
	}

	// Initialize web API
	webServer := webapi.NewServer(repo, cfg, logger)

	// Set up WebSocket callbacks for all clients
	for accountID, client := range binanceClients {
		accountID := accountID // capture variable
		client.SetOrderUpdateCallback(func(update *binance.OrderUpdate) {
			if executor != nil {
				executor.HandleOrderUpdate(update)
			}

			// Broadcast to web clients
			if webServer != nil {
				// Convert and broadcast
			}
		})
	}

	engine := &Engine{
		parser:         parser,
		executor:       executor,
		binanceClients: binanceClients,
		repo:           repo,
		webapi:         webServer,
		config:         cfg,
		logger:         logger,
	}

	return engine, nil
}

// Start starts the trading engine
func (e *Engine) Start() error {
	if !e.config.Trading.Enabled {
		e.logger.Info("Trading engine is disabled")
		return nil
	}

	e.logger.Info("Starting trading engine...")

	// Start Binance user data streams for all accounts
	for accountID, client := range e.binanceClients {
		listenKey, err := client.StartUserDataStream()
		if err != nil {
			e.logger.Errorf("Failed to start user data stream for account %d: %v", accountID, err)
			continue
		}

		if err := client.ConnectUserDataStream(listenKey); err != nil {
			e.logger.Errorf("Failed to connect to user data stream for account %d: %v", accountID, err)
			continue
		}

		e.logger.Infof("Started user data stream for account ID: %d", accountID)
	}

	e.logger.Info("Trading engine started successfully")

	// Start web API server
	if e.config.WebAPI.Enabled {
		go func() {
			if err := e.webapi.Start(); err != nil {
				e.logger.Errorf("Web API server error: %v", err)
			}
		}()
	}

	return nil
}

// ProcessMessage processes a Telegram message for trading signals
func (e *Engine) ProcessMessage(msg *models.Message) error {
	if !e.config.Trading.Enabled {
		return nil
	}

	// Try to parse signal
	signal, err := e.parser.Parse(msg)
	if err != nil {
		e.logger.Errorf("Failed to parse message: %v", err)
		return err
	}

	if signal == nil {
		// No signal found in this message
		return nil
	}

	// Validate symbol
	if !e.parser.IsValidSymbol(signal.Symbol) {
		e.logger.Warnf("Invalid symbol detected: %s", signal.Symbol)
		return nil
	}

	// Save signal to database
	if err := e.repo.SaveSignal(signal); err != nil {
		e.logger.Errorf("Failed to save signal: %v", err)
		return err
	}

	e.logger.WithFields(logrus.Fields{
		"signal_id": signal.ID,
		"symbol":    signal.Symbol,
	}).Info("New trading signal saved")

	// Execute the signal
	if err := e.executor.ExecuteSignal(signal); err != nil {
		e.logger.Errorf("Failed to execute signal: %v", err)

		// Update signal status to failed
		now := logrus.Now()
		e.repo.UpdateSignalStatus(signal.ID, "failed", &now, err.Error())

		// Broadcast error to web clients
		if e.webapi != nil {
			e.webapi.BroadcastUpdate("signal_error", map[string]interface{}{
				"signal_id": signal.ID,
				"symbol":    signal.Symbol,
				"error":     err.Error(),
			})
		}

		return err
	}

	// Update signal status to processed
	now := logrus.Now()
	e.repo.UpdateSignalStatus(signal.ID, "processed", &now, "")

	// Broadcast to web clients
	if e.webapi != nil {
		e.webapi.BroadcastUpdate("signal_executed", signal)
	}

	return nil
}

// Stop stops the trading engine
func (e *Engine) Stop() error {
	if !e.config.Trading.Enabled {
		return nil
	}

	e.logger.Info("Stopping trading engine...")

	if e.executor != nil {
		e.executor.Close()
	}

	// Close all Binance clients
	for accountID, client := range e.binanceClients {
		if err := client.Close(); err != nil {
			e.logger.Errorf("Error closing Binance client for account %d: %v", accountID, err)
		}
	}

	if e.webapi != nil {
		e.webapi.Stop()
	}

	e.logger.Info("Trading engine stopped")

	return nil
}

// GetStats returns trading statistics
func (e *Engine) GetStats() (*models.TradingStats, error) {
	return e.repo.GetTradingStats()
}

// GetOpenPositions returns all open positions
func (e *Engine) GetOpenPositions() ([]*models.Position, error) {
	return e.repo.GetOpenPositions()
}
