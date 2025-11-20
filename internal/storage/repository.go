package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"tdlib-go/pkg/models"
)

// Repository handles database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository instance
func NewRepository(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	repo := &Repository{db: db}

	// Run migrations
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

// migrate runs database migrations
func (r *Repository) migrate() error {
	schema, err := os.ReadFile("migrations/001_initial_schema.sql")
	if err != nil {
		// If migrations file doesn't exist, create schema inline
		return r.createSchemaInline()
	}

	if _, err := r.db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// createSchemaInline creates the database schema inline
func (r *Repository) createSchemaInline() error {
	schema := `
	CREATE TABLE IF NOT EXISTS binance_accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		api_key TEXT NOT NULL,
		api_secret TEXT NOT NULL,
		is_testnet BOOLEAN DEFAULT 0,
		is_active BOOLEAN DEFAULT 1,
		is_default BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_binance_accounts_is_active ON binance_accounts(is_active);
	CREATE INDEX IF NOT EXISTS idx_binance_accounts_is_default ON binance_accounts(is_default);

	CREATE TABLE IF NOT EXISTS channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL UNIQUE,
		username TEXT,
		title TEXT NOT NULL,
		is_active BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_channels_channel_id ON channels(channel_id);
	CREATE INDEX IF NOT EXISTS idx_channels_username ON channels(username);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id INTEGER NOT NULL,
		channel_id INTEGER NOT NULL,
		channel_name TEXT NOT NULL,
		sender_id INTEGER NOT NULL,
		sender_name TEXT,
		text TEXT,
		media_type TEXT,
		is_forwarded BOOLEAN DEFAULT 0,
		timestamp TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id, channel_id)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
	CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);

	CREATE TABLE IF NOT EXISTS signals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id INTEGER NOT NULL,
		channel_id INTEGER NOT NULL,
		symbol TEXT NOT NULL,
		raw_message TEXT NOT NULL,
		parsed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		processed_at TIMESTAMP,
		status TEXT DEFAULT 'pending',
		error TEXT,
		FOREIGN KEY (message_id, channel_id) REFERENCES messages(message_id, channel_id)
	);

	CREATE INDEX IF NOT EXISTS idx_signals_status ON signals(status);
	CREATE INDEX IF NOT EXISTS idx_signals_symbol ON signals(symbol);
	CREATE INDEX IF NOT EXISTS idx_signals_parsed_at ON signals(parsed_at);

	CREATE TABLE IF NOT EXISTS positions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		signal_id INTEGER,
		account_id INTEGER NOT NULL,
		symbol TEXT NOT NULL,
		side TEXT NOT NULL,
		entry_price REAL NOT NULL,
		quantity REAL NOT NULL,
		leverage INTEGER NOT NULL,
		take_profit_price REAL NOT NULL,
		stop_loss_price REAL NOT NULL,
		status TEXT DEFAULT 'open',
		opened_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		closed_at TIMESTAMP,
		exit_price REAL,
		pnl REAL,
		pnl_percent REAL,
		FOREIGN KEY (signal_id) REFERENCES signals(id),
		FOREIGN KEY (account_id) REFERENCES binance_accounts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_positions_status ON positions(status);
	CREATE INDEX IF NOT EXISTS idx_positions_symbol ON positions(symbol);
	CREATE INDEX IF NOT EXISTS idx_positions_opened_at ON positions(opened_at);
	CREATE INDEX IF NOT EXISTS idx_positions_account_id ON positions(account_id);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		position_id INTEGER NOT NULL,
		binance_order_id TEXT NOT NULL UNIQUE,
		symbol TEXT NOT NULL,
		side TEXT NOT NULL,
		type TEXT NOT NULL,
		orig_qty REAL NOT NULL,
		executed_qty REAL DEFAULT 0,
		price REAL NOT NULL,
		stop_price REAL,
		status TEXT DEFAULT 'NEW',
		time_in_force TEXT DEFAULT 'GTC',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		filled_at TIMESTAMP,
		canceled_at TIMESTAMP,
		order_purpose TEXT NOT NULL,
		FOREIGN KEY (position_id) REFERENCES positions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_orders_position_id ON orders(position_id);
	CREATE INDEX IF NOT EXISTS idx_orders_binance_order_id ON orders(binance_order_id);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
	CREATE INDEX IF NOT EXISTS idx_orders_order_purpose ON orders(order_purpose);

	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_settings_key ON settings(key);
	`

	if _, err := r.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

// SaveMessage saves a message to the database
func (r *Repository) SaveMessage(msg *models.Message) error {
	query := `
		INSERT OR IGNORE INTO messages
		(message_id, channel_id, channel_name, sender_id, sender_name, text, media_type, is_forwarded, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		msg.MessageID,
		msg.ChannelID,
		msg.ChannelName,
		msg.SenderID,
		msg.SenderName,
		msg.Text,
		msg.MediaType,
		msg.IsForwarded,
		msg.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetMessagesByChannel retrieves messages from a specific channel
func (r *Repository) GetMessagesByChannel(channelID int64, limit int) ([]*models.Message, error) {
	query := `
		SELECT id, message_id, channel_id, channel_name, sender_id, sender_name,
		       text, media_type, is_forwarded, timestamp, created_at
		FROM messages
		WHERE channel_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		msg := &models.Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.MessageID,
			&msg.ChannelID,
			&msg.ChannelName,
			&msg.SenderID,
			&msg.SenderName,
			&msg.Text,
			&msg.MediaType,
			&msg.IsForwarded,
			&msg.Timestamp,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveChannel saves a channel to the database
func (r *Repository) SaveChannel(channel *models.Channel) error {
	query := `
		INSERT INTO channels (channel_id, username, title, is_active)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(channel_id) DO UPDATE SET
			username = excluded.username,
			title = excluded.title,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(query,
		channel.ChannelID,
		channel.Username,
		channel.Title,
		channel.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to save channel: %w", err)
	}

	return nil
}

// GetChannel retrieves a channel by ID or username
func (r *Repository) GetChannel(identifier string) (*models.Channel, error) {
	query := `
		SELECT id, channel_id, username, title, is_active, created_at, updated_at
		FROM channels
		WHERE channel_id = ? OR username = ?
		LIMIT 1
	`

	channel := &models.Channel{}
	err := r.db.QueryRow(query, identifier, identifier).Scan(
		&channel.ID,
		&channel.ChannelID,
		&channel.Username,
		&channel.Title,
		&channel.IsActive,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return channel, nil
}

// GetAllChannels retrieves all active channels
func (r *Repository) GetAllChannels() ([]*models.Channel, error) {
	query := `
		SELECT id, channel_id, username, title, is_active, created_at, updated_at
		FROM channels
		WHERE is_active = 1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	var channels []*models.Channel
	for rows.Next() {
		channel := &models.Channel{}
		err := rows.Scan(
			&channel.ID,
			&channel.ChannelID,
			&channel.Username,
			&channel.Title,
			&channel.IsActive,
			&channel.CreatedAt,
			&channel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// DeactivateChannel marks a channel as inactive
func (r *Repository) DeactivateChannel(channelID int64) error {
	query := `UPDATE channels SET is_active = 0, updated_at = ? WHERE channel_id = ?`
	_, err := r.db.Exec(query, time.Now(), channelID)
	if err != nil {
		return fmt.Errorf("failed to deactivate channel: %w", err)
	}
	return nil
}

// GetDatabasePath returns the full path to the database file
func GetDatabasePath(dsn string) (string, error) {
	dir := filepath.Dir(dsn)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create database directory: %w", err)
		}
	}
	return dsn, nil
}

// ============= Binance Account Methods =============

// SaveAccount saves a Binance account to the database
func (r *Repository) SaveAccount(account *models.BinanceAccount) error {
	query := `
		INSERT INTO binance_accounts (name, api_key, api_secret, is_testnet, is_active, is_default)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query,
		account.Name,
		account.APIKey,
		account.APISecret,
		account.IsTestnet,
		account.IsActive,
		account.IsDefault,
	)
	if err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get account ID: %w", err)
	}
	account.ID = id

	// If this is the default account, unset other defaults
	if account.IsDefault {
		_, err = r.db.Exec(`UPDATE binance_accounts SET is_default = 0 WHERE id != ?`, account.ID)
		if err != nil {
			return fmt.Errorf("failed to update other accounts: %w", err)
		}
	}

	return nil
}

// UpdateAccount updates a Binance account
func (r *Repository) UpdateAccount(account *models.BinanceAccount) error {
	query := `
		UPDATE binance_accounts
		SET name = ?, api_key = ?, api_secret = ?, is_testnet = ?, is_active = ?, is_default = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query,
		account.Name,
		account.APIKey,
		account.APISecret,
		account.IsTestnet,
		account.IsActive,
		account.IsDefault,
		time.Now(),
		account.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	// If this is the default account, unset other defaults
	if account.IsDefault {
		_, err = r.db.Exec(`UPDATE binance_accounts SET is_default = 0 WHERE id != ?`, account.ID)
		if err != nil {
			return fmt.Errorf("failed to update other accounts: %w", err)
		}
	}

	return nil
}

// GetAccount retrieves an account by ID
func (r *Repository) GetAccount(id int64) (*models.BinanceAccount, error) {
	query := `
		SELECT id, name, api_key, api_secret, is_testnet, is_active, is_default, created_at, updated_at
		FROM binance_accounts
		WHERE id = ?
	`
	account := &models.BinanceAccount{}
	err := r.db.QueryRow(query, id).Scan(
		&account.ID,
		&account.Name,
		&account.APIKey,
		&account.APISecret,
		&account.IsTestnet,
		&account.IsActive,
		&account.IsDefault,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return account, nil
}

// GetAllAccounts retrieves all Binance accounts
func (r *Repository) GetAllAccounts() ([]*models.BinanceAccount, error) {
	query := `
		SELECT id, name, api_key, api_secret, is_testnet, is_active, is_default, created_at, updated_at
		FROM binance_accounts
		ORDER BY is_default DESC, name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*models.BinanceAccount
	for rows.Next() {
		account := &models.BinanceAccount{}
		err := rows.Scan(
			&account.ID,
			&account.Name,
			&account.APIKey,
			&account.APISecret,
			&account.IsTestnet,
			&account.IsActive,
			&account.IsDefault,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// GetActiveAccounts retrieves all active Binance accounts
func (r *Repository) GetActiveAccounts() ([]*models.BinanceAccount, error) {
	query := `
		SELECT id, name, api_key, api_secret, is_testnet, is_active, is_default, created_at, updated_at
		FROM binance_accounts
		WHERE is_active = 1
		ORDER BY is_default DESC, name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*models.BinanceAccount
	for rows.Next() {
		account := &models.BinanceAccount{}
		err := rows.Scan(
			&account.ID,
			&account.Name,
			&account.APIKey,
			&account.APISecret,
			&account.IsTestnet,
			&account.IsActive,
			&account.IsDefault,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// GetDefaultAccount retrieves the default Binance account
func (r *Repository) GetDefaultAccount() (*models.BinanceAccount, error) {
	query := `
		SELECT id, name, api_key, api_secret, is_testnet, is_active, is_default, created_at, updated_at
		FROM binance_accounts
		WHERE is_default = 1 AND is_active = 1
		LIMIT 1
	`
	account := &models.BinanceAccount{}
	err := r.db.QueryRow(query).Scan(
		&account.ID,
		&account.Name,
		&account.APIKey,
		&account.APISecret,
		&account.IsTestnet,
		&account.IsActive,
		&account.IsDefault,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default account: %w", err)
	}
	return account, nil
}

// DeleteAccount deletes a Binance account
func (r *Repository) DeleteAccount(id int64) error {
	// Check if account has open positions
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM positions WHERE account_id = ? AND status = 'open'`, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check positions: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete account with open positions")
	}

	query := `DELETE FROM binance_accounts WHERE id = ?`
	_, err = r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

// ============= Trading Methods =============

// SaveSignal saves a trading signal to the database
func (r *Repository) SaveSignal(signal *models.Signal) error {
	query := `
		INSERT INTO signals (message_id, channel_id, symbol, raw_message, parsed_at, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query,
		signal.MessageID,
		signal.ChannelID,
		signal.Symbol,
		signal.RawMessage,
		signal.ParsedAt,
		signal.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to save signal: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get signal ID: %w", err)
	}
	signal.ID = id

	return nil
}

// UpdateSignalStatus updates the status of a signal
func (r *Repository) UpdateSignalStatus(signalID int64, status string, processedAt *time.Time, errorMsg string) error {
	query := `UPDATE signals SET status = ?, processed_at = ?, error = ? WHERE id = ?`
	_, err := r.db.Exec(query, status, processedAt, errorMsg, signalID)
	if err != nil {
		return fmt.Errorf("failed to update signal status: %w", err)
	}
	return nil
}

// SavePosition saves a trading position to the database
func (r *Repository) SavePosition(pos *models.Position) error {
	query := `
		INSERT INTO positions (signal_id, account_id, symbol, side, entry_price, quantity, leverage,
		                       take_profit_price, stop_loss_price, status, opened_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query,
		pos.SignalID,
		pos.AccountID,
		pos.Symbol,
		pos.Side,
		pos.EntryPrice,
		pos.Quantity,
		pos.Leverage,
		pos.TakeProfitPrice,
		pos.StopLossPrice,
		pos.Status,
		pos.OpenedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get position ID: %w", err)
	}
	pos.ID = id

	return nil
}

// ClosePosition closes a position and calculates PnL
func (r *Repository) ClosePosition(positionID int64, exitPrice float64, closedAt time.Time) error {
	// First get the position to calculate PnL
	pos, err := r.GetPosition(positionID)
	if err != nil {
		return fmt.Errorf("failed to get position: %w", err)
	}

	// Calculate PnL
	var pnl, pnlPercent float64
	if pos.Side == "LONG" {
		pnl = (exitPrice - pos.EntryPrice) * pos.Quantity * float64(pos.Leverage)
		pnlPercent = ((exitPrice - pos.EntryPrice) / pos.EntryPrice) * 100 * float64(pos.Leverage)
	} else { // SHORT
		pnl = (pos.EntryPrice - exitPrice) * pos.Quantity * float64(pos.Leverage)
		pnlPercent = ((pos.EntryPrice - exitPrice) / pos.EntryPrice) * 100 * float64(pos.Leverage)
	}

	query := `
		UPDATE positions
		SET status = 'closed', exit_price = ?, closed_at = ?, pnl = ?, pnl_percent = ?
		WHERE id = ?
	`
	_, err = r.db.Exec(query, exitPrice, closedAt, pnl, pnlPercent, positionID)
	if err != nil {
		return fmt.Errorf("failed to close position: %w", err)
	}

	return nil
}

// GetPosition retrieves a position by ID
func (r *Repository) GetPosition(positionID int64) (*models.Position, error) {
	query := `
		SELECT id, signal_id, account_id, symbol, side, entry_price, quantity, leverage,
		       take_profit_price, stop_loss_price, status, opened_at, closed_at,
		       exit_price, pnl, pnl_percent
		FROM positions
		WHERE id = ?
	`
	pos := &models.Position{}
	err := r.db.QueryRow(query, positionID).Scan(
		&pos.ID,
		&pos.SignalID,
		&pos.AccountID,
		&pos.Symbol,
		&pos.Side,
		&pos.EntryPrice,
		&pos.Quantity,
		&pos.Leverage,
		&pos.TakeProfitPrice,
		&pos.StopLossPrice,
		&pos.Status,
		&pos.OpenedAt,
		&pos.ClosedAt,
		&pos.ExitPrice,
		&pos.PnL,
		&pos.PnLPercent,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}
	return pos, nil
}

// GetOpenPositions retrieves all open positions
func (r *Repository) GetOpenPositions() ([]*models.Position, error) {
	query := `
		SELECT id, signal_id, account_id, symbol, side, entry_price, quantity, leverage,
		       take_profit_price, stop_loss_price, status, opened_at, closed_at,
		       exit_price, pnl, pnl_percent
		FROM positions
		WHERE status = 'open'
		ORDER BY opened_at DESC
	`
	return r.queryPositions(query)
}

// GetAllPositions retrieves all positions with optional limit
func (r *Repository) GetAllPositions(limit int) ([]*models.Position, error) {
	query := `
		SELECT id, signal_id, account_id, symbol, side, entry_price, quantity, leverage,
		       take_profit_price, stop_loss_price, status, opened_at, closed_at,
		       exit_price, pnl, pnl_percent
		FROM positions
		ORDER BY opened_at DESC
		LIMIT ?
	`
	return r.queryPositions(query, limit)
}

// queryPositions is a helper function to query positions
func (r *Repository) queryPositions(query string, args ...interface{}) ([]*models.Position, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()

	var positions []*models.Position
	for rows.Next() {
		pos := &models.Position{}
		err := rows.Scan(
			&pos.ID,
			&pos.SignalID,
			&pos.AccountID,
			&pos.Symbol,
			&pos.Side,
			&pos.EntryPrice,
			&pos.Quantity,
			&pos.Leverage,
			&pos.TakeProfitPrice,
			&pos.StopLossPrice,
			&pos.Status,
			&pos.OpenedAt,
			&pos.ClosedAt,
			&pos.ExitPrice,
			&pos.PnL,
			&pos.PnLPercent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

// SaveOrder saves an order to the database
func (r *Repository) SaveOrder(order *models.Order) error {
	query := `
		INSERT INTO orders (position_id, binance_order_id, symbol, side, type, orig_qty,
		                   executed_qty, price, stop_price, status, time_in_force, order_purpose)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query,
		order.PositionID,
		order.BinanceOrderID,
		order.Symbol,
		order.Side,
		order.Type,
		order.OrigQty,
		order.ExecutedQty,
		order.Price,
		order.StopPrice,
		order.Status,
		order.TimeInForce,
		order.OrderPurpose,
	)
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get order ID: %w", err)
	}
	order.ID = id

	return nil
}

// UpdateOrderStatus updates the status of an order
func (r *Repository) UpdateOrderStatus(binanceOrderID string, status string, executedQty float64) error {
	now := time.Now()
	var filledAt, canceledAt *time.Time

	if status == "FILLED" {
		filledAt = &now
	} else if status == "CANCELED" || status == "EXPIRED" {
		canceledAt = &now
	}

	query := `
		UPDATE orders
		SET status = ?, executed_qty = ?, updated_at = ?, filled_at = ?, canceled_at = ?
		WHERE binance_order_id = ?
	`
	_, err := r.db.Exec(query, status, executedQty, now, filledAt, canceledAt, binanceOrderID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// GetOrdersByPosition retrieves all orders for a position
func (r *Repository) GetOrdersByPosition(positionID int64) ([]*models.Order, error) {
	query := `
		SELECT id, position_id, binance_order_id, symbol, side, type, orig_qty,
		       executed_qty, price, stop_price, status, time_in_force, created_at,
		       updated_at, filled_at, canceled_at, order_purpose
		FROM orders
		WHERE position_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(query, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		err := rows.Scan(
			&order.ID,
			&order.PositionID,
			&order.BinanceOrderID,
			&order.Symbol,
			&order.Side,
			&order.Type,
			&order.OrigQty,
			&order.ExecutedQty,
			&order.Price,
			&order.StopPrice,
			&order.Status,
			&order.TimeInForce,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.FilledAt,
			&order.CanceledAt,
			&order.OrderPurpose,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetTradingStats calculates trading statistics
func (r *Repository) GetTradingStats() (*models.TradingStats, error) {
	stats := &models.TradingStats{}

	// Count positions
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM positions WHERE status = 'closed'
	`).Scan(&stats.TotalTrades)
	if err != nil {
		return nil, fmt.Errorf("failed to count total trades: %w", err)
	}

	// Count winning/losing trades
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM positions WHERE status = 'closed' AND pnl > 0
	`).Scan(&stats.WinningTrades)
	if err != nil {
		return nil, fmt.Errorf("failed to count winning trades: %w", err)
	}

	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM positions WHERE status = 'closed' AND pnl <= 0
	`).Scan(&stats.LosingTrades)
	if err != nil {
		return nil, fmt.Errorf("failed to count losing trades: %w", err)
	}

	// Calculate total PnL
	err = r.db.QueryRow(`
		SELECT COALESCE(SUM(pnl), 0) FROM positions WHERE status = 'closed'
	`).Scan(&stats.TotalPnL)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total PnL: %w", err)
	}

	// Calculate average win/loss
	err = r.db.QueryRow(`
		SELECT COALESCE(AVG(pnl), 0) FROM positions WHERE status = 'closed' AND pnl > 0
	`).Scan(&stats.AverageWin)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate average win: %w", err)
	}

	err = r.db.QueryRow(`
		SELECT COALESCE(AVG(pnl), 0) FROM positions WHERE status = 'closed' AND pnl <= 0
	`).Scan(&stats.AverageLoss)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate average loss: %w", err)
	}

	// Find largest win/loss
	err = r.db.QueryRow(`
		SELECT COALESCE(MAX(pnl), 0) FROM positions WHERE status = 'closed'
	`).Scan(&stats.LargestWin)
	if err != nil {
		return nil, fmt.Errorf("failed to find largest win: %w", err)
	}

	err = r.db.QueryRow(`
		SELECT COALESCE(MIN(pnl), 0) FROM positions WHERE status = 'closed'
	`).Scan(&stats.LargestLoss)
	if err != nil {
		return nil, fmt.Errorf("failed to find largest loss: %w", err)
	}

	// Count open positions
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM positions WHERE status = 'open'
	`).Scan(&stats.OpenPositions)
	if err != nil {
		return nil, fmt.Errorf("failed to count open positions: %w", err)
	}

	// Calculate win rate
	if stats.TotalTrades > 0 {
		stats.WinRate = (float64(stats.WinningTrades) / float64(stats.TotalTrades)) * 100
	}

	return stats, nil
}

// SaveSetting saves or updates a setting
func (r *Repository) SaveSetting(key, value string) error {
	query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to save setting: %w", err)
	}

	return nil
}

// GetSetting retrieves a setting value by key
func (r *Repository) GetSetting(key string) (string, error) {
	var value string
	query := `SELECT value FROM settings WHERE key = ?`

	err := r.db.QueryRow(query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}

	return value, nil
}

// GetAllSettings retrieves all settings as a map
func (r *Repository) GetAllSettings() (map[string]string, error) {
	query := `SELECT key, value FROM settings`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings[key] = value
	}

	return settings, nil
}
