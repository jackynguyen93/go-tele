package trading

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"tdlib-go/internal/binance"
	"tdlib-go/internal/config"
	"tdlib-go/internal/storage"
	"tdlib-go/pkg/models"
)

// OrderExecutor handles order execution and management
type OrderExecutor struct {
	binanceClient *binance.Client
	repo          *storage.Repository
	config        *config.Config
	logger        *logrus.Logger
	accountID     int64 // Binance account ID

	// Async logging channel
	logQueue chan *LogEntry

	// Active positions tracking
	activePositions map[int64]*PositionTracker
	positionsMu     sync.RWMutex

	// Order timeout tracking
	pendingOrders map[string]*OrderTimeout
	ordersMu      sync.RWMutex
}

// LogEntry represents an entry to be logged asynchronously
type LogEntry struct {
	Type string
	Data interface{}
}

// PositionTracker tracks an active position
type PositionTracker struct {
	PositionID      int64
	Symbol          string
	EntryOrderID    string
	TPOrderID       string
	SLOrderID       string
	EntryFilled     bool
	TPFilled        bool
	SLFilled        bool
	CreatedAt       time.Time
}

// OrderTimeout tracks pending orders for timeout
type OrderTimeout struct {
	OrderID     string
	PositionID  int64
	Symbol      string
	OrderType   string
	CreatedAt   time.Time
	TimeoutDuration time.Duration
}

// NewOrderExecutor creates a new order executor
func NewOrderExecutor(binanceClient *binance.Client, repo *storage.Repository, cfg *config.Config, logger *logrus.Logger) *OrderExecutor {
	executor := &OrderExecutor{
		binanceClient:   binanceClient,
		repo:            repo,
		config:          cfg,
		logger:          logger,
		logQueue:        make(chan *LogEntry, 1000),
		activePositions: make(map[int64]*PositionTracker),
		pendingOrders:   make(map[string]*OrderTimeout),
	}

	// Start async logger
	go executor.runAsyncLogger()

	// Start order timeout monitor
	go executor.monitorOrderTimeouts()

	return executor
}

// ExecuteSignal executes a trading signal
func (e *OrderExecutor) ExecuteSignal(signal *models.Signal) error {
	if e.config.Trading.DryRun {
		e.logger.Warnf("DRY RUN: Would execute signal for %s", signal.Symbol)
		return nil
	}

	// Check max positions
	if err := e.checkMaxPositions(); err != nil {
		return err
	}

	// Get current price
	ticker, err := e.binanceClient.GetSymbolPriceTicker(signal.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get price for %s: %w", signal.Symbol, err)
	}

	entryPrice, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		return fmt.Errorf("failed to parse price: %w", err)
	}

	// Calculate order parameters
	leverage := e.config.Trading.Leverage
	orderAmount := e.config.Trading.OrderAmount
	targetPercent := e.config.Trading.TargetPercent
	stopLossPercent := e.config.Trading.StopLossPercent

	// Calculate prices
	takeProfitPrice := entryPrice * (1 + targetPercent*float64(leverage))
	stopLossPrice := entryPrice * (1 - stopLossPercent*float64(leverage))

	// Calculate quantity based on order amount
	quantity := orderAmount / entryPrice

	e.logger.WithFields(logrus.Fields{
		"symbol":            signal.Symbol,
		"entry_price":       entryPrice,
		"take_profit_price": takeProfitPrice,
		"stop_loss_price":   stopLossPrice,
		"quantity":          quantity,
		"leverage":          leverage,
	}).Info("Executing trading signal")

	// Set leverage
	if err := e.binanceClient.SetLeverage(signal.Symbol, leverage); err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	// Set margin type to CROSSED
	if err := e.binanceClient.SetMarginType(signal.Symbol, "CROSSED"); err != nil {
		return fmt.Errorf("failed to set margin type: %w", err)
	}

	// Create position record FIRST (before orders)
	position := &models.Position{
		SignalID:        signal.ID,
		AccountID:       e.accountID,
		Symbol:          signal.Symbol,
		Side:            "LONG",
		EntryPrice:      entryPrice,
		Quantity:        quantity,
		Leverage:        leverage,
		TakeProfitPrice: takeProfitPrice,
		StopLossPrice:   stopLossPrice,
		Status:          "open",
		OpenedAt:        time.Now(),
	}

	// Save position to get ID
	if err := e.repo.SavePosition(position); err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	// Execute 3 orders in parallel for speed
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	var entryResp, tpResp, slResp *binance.OrderResponse
	var entryErr, tpErr, slErr error

	// 1. Entry order (MARKET BUY)
	wg.Add(1)
	go func() {
		defer wg.Done()
		entryResp, entryErr = e.binanceClient.PlaceOrder(&binance.NewOrder{
			Symbol:   signal.Symbol,
			Side:     "BUY",
			Type:     "MARKET",
			Quantity: quantity,
		})
		if entryErr != nil {
			errChan <- fmt.Errorf("entry order failed: %w", entryErr)
		}
	}()

	// 2. Take Profit order (TAKE_PROFIT_MARKET)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tpResp, tpErr = e.binanceClient.PlaceOrder(&binance.NewOrder{
			Symbol:     signal.Symbol,
			Side:       "SELL",
			Type:       "TAKE_PROFIT_MARKET",
			StopPrice:  takeProfitPrice,
			Quantity:   quantity,
			ReduceOnly: true,
		})
		if tpErr != nil {
			errChan <- fmt.Errorf("take profit order failed: %w", tpErr)
		}
	}()

	// 3. Stop Loss order (STOP_MARKET)
	wg.Add(1)
	go func() {
		defer wg.Done()
		slResp, slErr = e.binanceClient.PlaceOrder(&binance.NewOrder{
			Symbol:     signal.Symbol,
			Side:       "SELL",
			Type:       "STOP_MARKET",
			StopPrice:  stopLossPrice,
			Quantity:   quantity,
			ReduceOnly: true,
		})
		if slErr != nil {
			errChan <- fmt.Errorf("stop loss order failed: %w", slErr)
		}
	}()

	// Wait for all orders to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// If entry failed, cancel TP/SL if they were placed
		if entryErr != nil {
			e.logger.Error("Entry order failed, canceling TP/SL orders")
			if tpResp != nil {
				e.binanceClient.CancelOrder(signal.Symbol, tpResp.OrderID)
			}
			if slResp != nil {
				e.binanceClient.CancelOrder(signal.Symbol, slResp.OrderID)
			}
			return fmt.Errorf("order execution failed: %v", errors)
		}
	}

	// Log orders asynchronously (non-blocking)
	if entryResp != nil {
		e.asyncLogOrder(position.ID, entryResp, "entry")
	}
	if tpResp != nil {
		e.asyncLogOrder(position.ID, tpResp, "take_profit")
		// Add to timeout tracker
		e.addOrderTimeout(strconv.FormatInt(tpResp.OrderID, 10), position.ID, signal.Symbol, "take_profit")
	}
	if slResp != nil {
		e.asyncLogOrder(position.ID, slResp, "stop_loss")
		// Add to timeout tracker
		e.addOrderTimeout(strconv.FormatInt(slResp.OrderID, 10), position.ID, signal.Symbol, "stop_loss")
	}

	// Track position
	tracker := &PositionTracker{
		PositionID:   position.ID,
		Symbol:       signal.Symbol,
		EntryFilled:  entryResp.Status == "FILLED",
		CreatedAt:    time.Now(),
	}
	if tpResp != nil {
		tracker.TPOrderID = strconv.FormatInt(tpResp.OrderID, 10)
	}
	if slResp != nil {
		tracker.SLOrderID = strconv.FormatInt(slResp.OrderID, 10)
	}

	e.positionsMu.Lock()
	e.activePositions[position.ID] = tracker
	e.positionsMu.Unlock()

	e.logger.WithFields(logrus.Fields{
		"position_id": position.ID,
		"symbol":      signal.Symbol,
		"entry_status": entryResp.Status,
	}).Info("Signal executed successfully")

	return nil
}

// asyncLogOrder logs an order asynchronously
func (e *OrderExecutor) asyncLogOrder(positionID int64, orderResp *binance.OrderResponse, purpose string) {
	price, _ := strconv.ParseFloat(orderResp.Price, 64)
	origQty, _ := strconv.ParseFloat(orderResp.OrigQty, 64)
	executedQty, _ := strconv.ParseFloat(orderResp.ExecutedQty, 64)
	var stopPrice *float64
	if orderResp.StopPrice != "" {
		sp, _ := strconv.ParseFloat(orderResp.StopPrice, 64)
		stopPrice = &sp
	}

	order := &models.Order{
		PositionID:      positionID,
		BinanceOrderID:  strconv.FormatInt(orderResp.OrderID, 10),
		Symbol:          orderResp.Symbol,
		Side:            orderResp.Side,
		Type:            orderResp.Type,
		OrigQty:         origQty,
		ExecutedQty:     executedQty,
		Price:           price,
		StopPrice:       stopPrice,
		Status:          orderResp.Status,
		TimeInForce:     orderResp.TimeInForce,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		OrderPurpose:    purpose,
	}

	e.logQueue <- &LogEntry{
		Type: "order",
		Data: order,
	}
}

// runAsyncLogger processes log entries asynchronously
func (e *OrderExecutor) runAsyncLogger() {
	for entry := range e.logQueue {
		switch entry.Type {
		case "order":
			order := entry.Data.(*models.Order)
			if err := e.repo.SaveOrder(order); err != nil {
				e.logger.Errorf("Failed to save order asynchronously: %v", err)
			}
		case "position_update":
			// Handle position updates
		}
	}
}

// checkMaxPositions checks if we can open a new position
func (e *OrderExecutor) checkMaxPositions() error {
	openPositions, err := e.repo.GetOpenPositions()
	if err != nil {
		return fmt.Errorf("failed to get open positions: %w", err)
	}

	if len(openPositions) >= e.config.Trading.MaxPositions {
		return fmt.Errorf("max positions reached (%d/%d)", len(openPositions), e.config.Trading.MaxPositions)
	}

	return nil
}

// addOrderTimeout adds an order to the timeout tracker
func (e *OrderExecutor) addOrderTimeout(orderID string, positionID int64, symbol string, orderType string) {
	timeout := &OrderTimeout{
		OrderID:         orderID,
		PositionID:      positionID,
		Symbol:          symbol,
		OrderType:       orderType,
		CreatedAt:       time.Now(),
		TimeoutDuration: time.Duration(e.config.Trading.OrderTimeout) * time.Second,
	}

	e.ordersMu.Lock()
	e.pendingOrders[orderID] = timeout
	e.ordersMu.Unlock()
}

// monitorOrderTimeouts monitors and cancels timed-out orders
func (e *OrderExecutor) monitorOrderTimeouts() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		e.ordersMu.Lock()
		for orderID, timeout := range e.pendingOrders {
			if time.Since(timeout.CreatedAt) > timeout.TimeoutDuration {
				// Timeout reached, cancel order
				e.logger.Infof("Order %s timed out after %v, canceling...", orderID, timeout.TimeoutDuration)

				binanceOrderID, _ := strconv.ParseInt(orderID, 10, 64)
				_, err := e.binanceClient.CancelOrder(timeout.Symbol, binanceOrderID)
				if err != nil {
					e.logger.Errorf("Failed to cancel timed-out order %s: %v", orderID, err)
				} else {
					// Update order status asynchronously
					go e.repo.UpdateOrderStatus(orderID, "CANCELED", 0)
				}

				// Remove from pending
				delete(e.pendingOrders, orderID)
			}
		}
		e.ordersMu.Unlock()
	}
}

// HandleOrderUpdate handles order updates from WebSocket
func (e *OrderExecutor) HandleOrderUpdate(update *binance.OrderUpdate) {
	orderID := strconv.FormatInt(update.Order.OrderID, 10)

	e.logger.WithFields(logrus.Fields{
		"order_id": orderID,
		"symbol":   update.Order.Symbol,
		"status":   update.Order.OrderStatus,
		"type":     update.Order.ExecutionType,
	}).Info("Order update received")

	// Update order status asynchronously
	executedQty, _ := strconv.ParseFloat(update.Order.FilledQty, 64)
	go e.repo.UpdateOrderStatus(orderID, update.Order.OrderStatus, executedQty)

	// Remove from pending if filled or canceled
	if update.Order.OrderStatus == "FILLED" || update.Order.OrderStatus == "CANCELED" || update.Order.OrderStatus == "EXPIRED" {
		e.ordersMu.Lock()
		timeout, exists := e.pendingOrders[orderID]
		if exists {
			delete(e.pendingOrders, orderID)

			// Update position tracker
			e.positionsMu.Lock()
			if tracker, ok := e.activePositions[timeout.PositionID]; ok {
				if timeout.OrderType == "take_profit" {
					tracker.TPFilled = (update.Order.OrderStatus == "FILLED")
					if tracker.TPFilled {
						// Close position
						realizedProfit, _ := strconv.ParseFloat(update.Order.RealizedProfit, 64)
						avgPrice, _ := strconv.ParseFloat(update.Order.AvgPrice, 64)
						go e.closePosition(timeout.PositionID, avgPrice, realizedProfit)
					}
				} else if timeout.OrderType == "stop_loss" {
					tracker.SLFilled = (update.Order.OrderStatus == "FILLED")
					if tracker.SLFilled {
						// Close position
						realizedProfit, _ := strconv.ParseFloat(update.Order.RealizedProfit, 64)
						avgPrice, _ := strconv.ParseFloat(update.Order.AvgPrice, 64)
						go e.closePosition(timeout.PositionID, avgPrice, realizedProfit)
					}
				}
			}
			e.positionsMu.Unlock()
		}
		e.ordersMu.Unlock()
	}
}

// closePosition closes a position
func (e *OrderExecutor) closePosition(positionID int64, exitPrice float64, realizedPnL float64) {
	if err := e.repo.ClosePosition(positionID, exitPrice, time.Now()); err != nil {
		e.logger.Errorf("Failed to close position %d: %v", positionID, err)
		return
	}

	// Remove from active positions
	e.positionsMu.Lock()
	delete(e.activePositions, positionID)
	e.positionsMu.Unlock()

	e.logger.WithFields(logrus.Fields{
		"position_id": positionID,
		"exit_price":  exitPrice,
		"pnl":         realizedPnL,
	}).Info("Position closed")
}

// Close shuts down the executor
func (e *OrderExecutor) Close() {
	close(e.logQueue)
}
