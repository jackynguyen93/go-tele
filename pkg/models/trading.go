package models

import "time"

// BinanceAccount represents a Binance account configuration
type BinanceAccount struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`                 // Friendly name for the account
	APIKey    string    `db:"api_key" json:"api_key"`           // Encrypted in production
	APISecret string    `db:"api_secret" json:"api_secret"`     // Encrypted in production
	IsTestnet bool      `db:"is_testnet" json:"is_testnet"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	IsDefault bool      `db:"is_default" json:"is_default"`     // Default account for new trades
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Signal represents a parsed trading signal from Telegram
type Signal struct {
	ID          int64     `db:"id"`
	MessageID   int64     `db:"message_id"`
	ChannelID   int64     `db:"channel_id"`
	Symbol      string    `db:"symbol"`
	RawMessage  string    `db:"raw_message"`
	ParsedAt    time.Time `db:"parsed_at"`
	ProcessedAt *time.Time `db:"processed_at"`
	Status      string    `db:"status"` // pending, processed, failed
	Error       string    `db:"error"`
}

// Position represents an open trading position
type Position struct {
	ID              int64      `db:"id" json:"id"`
	SignalID        int64      `db:"signal_id" json:"signal_id"`
	AccountID       int64      `db:"account_id" json:"account_id"`      // Which Binance account
	Symbol          string     `db:"symbol" json:"symbol"`
	Side            string     `db:"side" json:"side"` // LONG, SHORT
	EntryPrice      float64    `db:"entry_price" json:"entry_price"`
	Quantity        float64    `db:"quantity" json:"quantity"`
	Leverage        int        `db:"leverage" json:"leverage"`
	TakeProfitPrice float64    `db:"take_profit_price" json:"take_profit_price"`
	StopLossPrice   float64    `db:"stop_loss_price" json:"stop_loss_price"`
	Status          string     `db:"status" json:"status"` // open, closed, cancelled
	OpenedAt        time.Time  `db:"opened_at" json:"opened_at"`
	ClosedAt        *time.Time `db:"closed_at" json:"closed_at"`
	ExitPrice       *float64   `db:"exit_price" json:"exit_price"`
	PnL             *float64   `db:"pnl" json:"pnl"`
	PnLPercent      *float64   `db:"pnl_percent" json:"pnl_percent"`
}

// Order represents a Binance order
type Order struct {
	ID              int64      `db:"id" json:"id"`
	PositionID      int64      `db:"position_id" json:"position_id"`
	BinanceOrderID  string     `db:"binance_order_id" json:"binance_order_id"`
	Symbol          string     `db:"symbol" json:"symbol"`
	Side            string     `db:"side" json:"side"` // BUY, SELL
	Type            string     `db:"type" json:"type"` // MARKET, LIMIT, STOP_MARKET, TAKE_PROFIT_MARKET
	OrigQty         float64    `db:"orig_qty" json:"orig_qty"`
	ExecutedQty     float64    `db:"executed_qty" json:"executed_qty"`
	Price           float64    `db:"price" json:"price"`
	StopPrice       *float64   `db:"stop_price" json:"stop_price"`
	Status          string     `db:"status" json:"status"` // NEW, FILLED, PARTIALLY_FILLED, CANCELED, EXPIRED
	TimeInForce     string     `db:"time_in_force" json:"time_in_force"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
	FilledAt        *time.Time `db:"filled_at" json:"filled_at"`
	CanceledAt      *time.Time `db:"canceled_at" json:"canceled_at"`
	OrderPurpose    string     `db:"order_purpose" json:"order_purpose"` // entry, take_profit, stop_loss
}

// TradingStats represents trading statistics
type TradingStats struct {
	TotalTrades     int
	WinningTrades   int
	LosingTrades    int
	TotalPnL        float64
	WinRate         float64
	AverageWin      float64
	AverageLoss     float64
	LargestWin      float64
	LargestLoss     float64
	OpenPositions   int
}
