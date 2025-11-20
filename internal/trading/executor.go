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

	e.logger.Infof("Position created: ID=%d, Symbol=%s, Status=%s", position.ID, position.Symbol, position.Status)

	// DRY RUN MODE: Create simulated orders without actually placing them
	if e.config.Trading.DryRun {
		e.logger.Warnf("DRY RUN: Simulating orders for %s (Position ID: %d)", signal.Symbol, position.ID)

		// Create simulated entry order
		entryOrder := &models.Order{
			PositionID:      position.ID,
			BinanceOrderID:  fmt.Sprintf("DRY_ENTRY_%d_%d", position.ID, time.Now().Unix()),
			Symbol:          signal.Symbol,
			Side:            "BUY",
			Type:            "MARKET",
			OrigQty:         quantity,
			ExecutedQty:     quantity,
			Price:           entryPrice,
			Status:          "FILLED",
			TimeInForce:     "GTC",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			OrderPurpose:    "entry",
		}
		now := time.Now()
		entryOrder.FilledAt = &now
		if err := e.repo.SaveOrder(entryOrder); err != nil {
			e.logger.Errorf("Failed to save simulated entry order: %v", err)
		}

		// Create simulated take profit order
		tpOrder := &models.Order{
			PositionID:      position.ID,
			BinanceOrderID:  fmt.Sprintf("DRY_TP_%d_%d", position.ID, time.Now().Unix()),
			Symbol:          signal.Symbol,
			Side:            "SELL",
			Type:            "TAKE_PROFIT_MARKET",
			OrigQty:         quantity,
			ExecutedQty:     0,
			Price:           takeProfitPrice,
			StopPrice:       &takeProfitPrice,
			Status:          "NEW",
			TimeInForce:     "GTC",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			OrderPurpose:    "take_profit",
		}
		if err := e.repo.SaveOrder(tpOrder); err != nil {
			e.logger.Errorf("Failed to save simulated TP order: %v", err)
		}

		// Create simulated stop loss order
		slOrder := &models.Order{
			PositionID:      position.ID,
			BinanceOrderID:  fmt.Sprintf("DRY_SL_%d_%d", position.ID, time.Now().Unix()),
			Symbol:          signal.Symbol,
			Side:            "SELL",
			Type:            "STOP_MARKET",
			OrigQty:         quantity,
			ExecutedQty:     0,
			Price:           stopLossPrice,
			StopPrice:       &stopLossPrice,
			Status:          "NEW",
			TimeInForce:     "GTC",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			OrderPurpose:    "stop_loss",
		}
		if err := e.repo.SaveOrder(slOrder); err != nil {
			e.logger.Errorf("Failed to save simulated SL order: %v", err)
		}

		e.logger.Infof("DRY RUN: Created simulated position with 3 orders for %s", signal.Symbol)
		return nil
	}

	// REAL MODE: Set leverage and margin type, then place actual orders
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
