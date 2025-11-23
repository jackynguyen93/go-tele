package trading

import (
	"fmt"
	"math"
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

	// Order timeout tracking
	pendingOrders map[string]*OrderTimeout
	ordersMu      sync.RWMutex

	// Recent signal tracking (prevent duplicates within 48h)
	recentSignals map[string]time.Time
	signalsMu     sync.RWMutex
}

// LogEntry represents an entry to be logged asynchronously
type LogEntry struct {
	Type string
	Data interface{}
}

// OrderTimeout tracks pending orders for timeout
type OrderTimeout struct {
	OrderID         string
	Symbol          string
	OrderType       string
	Quantity        float64 // Position quantity for closing when timeout
	CreatedAt       time.Time
	TimeoutDuration time.Duration
}

// NewOrderExecutor creates a new order executor
func NewOrderExecutor(binanceClient *binance.Client, repo *storage.Repository, cfg *config.Config, logger *logrus.Logger) *OrderExecutor {
	executor := &OrderExecutor{
		binanceClient: binanceClient,
		repo:          repo,
		config:        cfg,
		logger:        logger,
		logQueue:      make(chan *LogEntry, 1000),
		pendingOrders: make(map[string]*OrderTimeout),
		recentSignals: make(map[string]time.Time),
	}

	// Start async logger
	go executor.runAsyncLogger()

	// Start order timeout monitor
	go executor.monitorOrderTimeouts()

	// Start signal cleanup monitor (remove signals older than 48h)
	go executor.cleanupOldSignals()

	return executor
}

// ExecuteSignal executes a trading signal with account-specific configuration
func (e *OrderExecutor) ExecuteSignal(signal *models.Signal, account *models.BinanceAccount) error {
	// Check if we've recently executed this signal (within 48 hours)
	e.signalsMu.RLock()
	lastExecuted, exists := e.recentSignals[signal.Symbol]
	e.signalsMu.RUnlock()

	if exists {
		timeSince := time.Since(lastExecuted)
		if timeSince < 48*time.Hour {
			e.logger.Warnf("Skipping duplicate signal for %s (last executed %v ago, cooldown: 48h)",
				signal.Symbol, timeSince.Round(time.Minute))
			return nil
		}
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

	// Use account-specific configuration
	leverage := account.Leverage
	orderAmount := account.OrderAmount
	targetPercent := account.TargetPercent
	stopLossPercent := account.StopLossPercent

	// Validate account configuration
	if leverage <= 0 || leverage > 125 {
		return fmt.Errorf("invalid leverage %d for account %s (must be between 1 and 125)", leverage, account.Name)
	}
	if orderAmount <= 0 {
		return fmt.Errorf("invalid order amount %.2f for account %s (must be greater than 0)", orderAmount, account.Name)
	}
	if targetPercent <= 0 {
		return fmt.Errorf("invalid target percent %.4f for account %s (must be greater than 0)", targetPercent, account.Name)
	}
	if stopLossPercent <= 0 {
		return fmt.Errorf("invalid stop loss percent %.4f for account %s (must be greater than 0)", stopLossPercent, account.Name)
	}

	// Calculate prices (divide by leverage since price movement is amplified)
	// e.g., 20% target with 10x leverage = 2% price change needed
	takeProfitPrice := entryPrice * (1 + targetPercent/float64(leverage))
	stopLossPrice := entryPrice * (1 - stopLossPercent/float64(leverage))

	// Calculate quantity based on order amount
	quantity := orderAmount / entryPrice

	// Get exchange info to determine precision
	exchangeInfo, err := e.binanceClient.GetExchangeInfo()
	if err != nil {
		return fmt.Errorf("failed to get exchange info: %w", err)
	}

	// Find symbol info
	var symbolInfo *binance.SymbolInfo
	for i := range exchangeInfo.Symbols {
		if exchangeInfo.Symbols[i].Symbol == signal.Symbol {
			symbolInfo = &exchangeInfo.Symbols[i]
			break
		}
	}

	if symbolInfo == nil {
		return fmt.Errorf("symbol %s not found in exchange info", signal.Symbol)
	}

	// Get filters for precise rounding
	var lotFilter, priceFilter, minNotionalFilter *binance.FilterInfo
	for i := range symbolInfo.Filters {
		filter := &symbolInfo.Filters[i]
		switch filter.FilterType {
		case "MARKET_LOT_SIZE":
			lotFilter = filter
		case "LOT_SIZE":
			if lotFilter == nil { // Prefer MARKET_LOT_SIZE over LOT_SIZE
				lotFilter = filter
			}
		case "PRICE_FILTER":
			priceFilter = filter
		case "MIN_NOTIONAL":
			minNotionalFilter = filter
		}
	}

	// Round quantity and prices using filter-based precision
	if lotFilter != nil && lotFilter.StepSize != "" {
		quantity = e.roundToStepSize(quantity, lotFilter.StepSize, lotFilter.MinQty, lotFilter.MaxQty)
	} else {
		// Fallback to precision-based rounding
		quantity = e.roundToPrecision(quantity, symbolInfo.QuantityPrecision)
	}

	if priceFilter != nil && priceFilter.TickSize != "" {
		takeProfitPrice = e.roundToStepSize(takeProfitPrice, priceFilter.TickSize, priceFilter.MinPrice, priceFilter.MaxPrice)
		stopLossPrice = e.roundToStepSize(stopLossPrice, priceFilter.TickSize, priceFilter.MinPrice, priceFilter.MaxPrice)
	} else {
		// Fallback to precision-based rounding
		takeProfitPrice = e.roundToPrecision(takeProfitPrice, symbolInfo.PricePrecision)
		stopLossPrice = e.roundToPrecision(stopLossPrice, symbolInfo.PricePrecision)
	}

	// Check and adjust for MIN_NOTIONAL requirement
	if minNotionalFilter != nil && minNotionalFilter.MinNotional != "" {
		minNotional, err := strconv.ParseFloat(minNotionalFilter.MinNotional, 64)
		if err == nil {
			notional := quantity * entryPrice
			if notional < minNotional {
				// Increase quantity to meet minimum notional
				quantity = minNotional / entryPrice

				// Re-apply step size rounding after adjustment
				if lotFilter != nil && lotFilter.StepSize != "" {
					quantity = e.roundToStepSize(quantity, lotFilter.StepSize, lotFilter.MinQty, lotFilter.MaxQty)
				}

				e.logger.Infof("Adjusted quantity to meet MIN_NOTIONAL (%.2f USD): %.8f", minNotional, quantity)
			}
		}
	}

	e.logger.WithFields(logrus.Fields{
		"symbol":            signal.Symbol,
		"entry_price":       entryPrice,
		"take_profit_price": takeProfitPrice,
		"stop_loss_price":   stopLossPrice,
		"quantity":          quantity,
		"leverage":          leverage,
	}).Info("Executing trading signal")

	// Set leverage and margin type
	if err := e.binanceClient.SetLeverage(signal.Symbol, leverage); err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	if err := e.binanceClient.SetMarginType(signal.Symbol, "CROSSED"); err != nil {
		return fmt.Errorf("failed to set margin type: %w", err)
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

	// Log order details
	if entryResp != nil {
		e.logger.WithFields(logrus.Fields{
			"order_id": entryResp.OrderID,
			"symbol":   entryResp.Symbol,
			"side":     entryResp.Side,
			"type":     entryResp.Type,
			"status":   entryResp.Status,
			"qty":      entryResp.OrigQty,
		}).Info("Entry order placed")
	}

	// Track TP/SL orders for timeout cancellation
	if tpResp != nil {
		e.logger.WithFields(logrus.Fields{
			"order_id":   tpResp.OrderID,
			"symbol":     tpResp.Symbol,
			"side":       tpResp.Side,
			"type":       tpResp.Type,
			"status":     tpResp.Status,
			"stop_price": tpResp.StopPrice,
		}).Info("Take profit order placed")

		// Add to timeout tracker with quantity for position closing
		e.addOrderTimeout(strconv.FormatInt(tpResp.OrderID, 10), signal.Symbol, "take_profit", quantity, account.OrderTimeout)
	}

	if slResp != nil {
		e.logger.WithFields(logrus.Fields{
			"order_id":   slResp.OrderID,
			"symbol":     slResp.Symbol,
			"side":       slResp.Side,
			"type":       slResp.Type,
			"status":     slResp.Status,
			"stop_price": slResp.StopPrice,
		}).Info("Stop loss order placed")

		// Add to timeout tracker with quantity for position closing
		e.addOrderTimeout(strconv.FormatInt(slResp.OrderID, 10), signal.Symbol, "stop_loss", quantity, account.OrderTimeout)
	}

	e.logger.WithFields(logrus.Fields{
		"symbol": signal.Symbol,
		"entry_status": entryResp.Status,
	}).Info("Signal executed successfully")

	// Record this signal to prevent duplicates within 48 hours
	e.signalsMu.Lock()
	e.recentSignals[signal.Symbol] = time.Now()
	e.signalsMu.Unlock()

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


// addOrderTimeout adds an order to the timeout tracker
func (e *OrderExecutor) addOrderTimeout(orderID string, symbol string, orderType string, quantity float64, timeoutSeconds int) {
	timeout := &OrderTimeout{
		OrderID:         orderID,
		Symbol:          symbol,
		OrderType:       orderType,
		Quantity:        quantity,
		CreatedAt:       time.Now(),
		TimeoutDuration: time.Duration(timeoutSeconds) * time.Second,
	}

	e.ordersMu.Lock()
	e.pendingOrders[orderID] = timeout
	e.ordersMu.Unlock()
}

// monitorOrderTimeouts monitors and cancels timed-out orders
func (e *OrderExecutor) monitorOrderTimeouts() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Track which positions we've already closed to avoid duplicate closes
	closedPositions := make(map[string]bool)

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
				}

				// Close the position if not already closed
				positionKey := timeout.Symbol
				if !closedPositions[positionKey] {
					e.logger.Infof("Closing open position for %s due to timeout", timeout.Symbol)

					// Place market sell order to close position
					_, err := e.binanceClient.PlaceOrder(&binance.NewOrder{
						Symbol:     timeout.Symbol,
						Side:       "SELL",
						Type:       "MARKET",
						Quantity:   timeout.Quantity,
						ReduceOnly: true,
					})

					if err != nil {
						e.logger.Errorf("Failed to close position for %s: %v", timeout.Symbol, err)
					} else {
						e.logger.Infof("Successfully closed position for %s (qty: %.8f)", timeout.Symbol, timeout.Quantity)
						closedPositions[positionKey] = true
					}
				}

				// Remove from pending
				delete(e.pendingOrders, orderID)
			}
		}
		e.ordersMu.Unlock()
	}
}

// cleanupOldSignals removes signals older than 48 hours from tracking
func (e *OrderExecutor) cleanupOldSignals() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		e.signalsMu.Lock()
		now := time.Now()
		for symbol, executedAt := range e.recentSignals {
			if now.Sub(executedAt) > 48*time.Hour {
				delete(e.recentSignals, symbol)
				e.logger.Debugf("Removed old signal tracking for %s (executed %v ago)", symbol, now.Sub(executedAt).Round(time.Hour))
			}
		}
		e.signalsMu.Unlock()
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

	// Remove from pending timeout tracker if filled or canceled
	if update.Order.OrderStatus == "FILLED" || update.Order.OrderStatus == "CANCELED" || update.Order.OrderStatus == "EXPIRED" {
		e.ordersMu.Lock()
		if _, exists := e.pendingOrders[orderID]; exists {
			delete(e.pendingOrders, orderID)
			e.logger.Infof("Removed order %s from timeout tracker (status: %s)", orderID, update.Order.OrderStatus)
		}
		e.ordersMu.Unlock()
	}
}

// Close shuts down the executor
func (e *OrderExecutor) Close() {
	close(e.logQueue)
}

// roundToPrecision rounds a number to the specified decimal precision
func (e *OrderExecutor) roundToPrecision(value float64, precision int) float64 {
	multiplier := float64(1)
	for i := 0; i < precision; i++ {
		multiplier *= 10
	}
	return float64(int64(value*multiplier)) / multiplier
}

// roundToStepSize rounds a value to the nearest valid multiple of stepSize
// and ensures it's within min/max bounds
func (e *OrderExecutor) roundToStepSize(value float64, stepSize, minQty, maxQty string) float64 {
	step, err := strconv.ParseFloat(stepSize, 64)
	if err != nil || step == 0 {
		e.logger.Warnf("Invalid stepSize: %s, using original value", stepSize)
		return value
	}

	min, _ := strconv.ParseFloat(minQty, 64)
	max, _ := strconv.ParseFloat(maxQty, 64)

	// Round to nearest multiple of stepSize
	rounded := math.Floor(value/step+0.5) * step

	// Ensure within bounds
	if min > 0 && rounded < min {
		rounded = min
	}
	if max > 0 && rounded > max {
		rounded = max
	}

	return rounded
}
