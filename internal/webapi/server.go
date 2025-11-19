package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"tdlib-go/internal/config"
	"tdlib-go/internal/storage"
	"tdlib-go/pkg/models"
)

// Server represents the web API server
type Server struct {
	router *mux.Router
	server *http.Server
	repo   *storage.Repository
	config *config.Config
	logger *logrus.Logger

	// WebSocket clients
	wsClients   map[*websocket.Conn]bool
	wsClientsMu sync.RWMutex
	wsBroadcast chan interface{}
	upgrader    websocket.Upgrader
}

// NewServer creates a new web API server
func NewServer(repo *storage.Repository, cfg *config.Config, logger *logrus.Logger) *Server {
	s := &Server{
		router:      mux.NewRouter(),
		repo:        repo,
		config:      cfg,
		logger:      logger,
		wsClients:   make(map[*websocket.Conn]bool),
		wsBroadcast: make(chan interface{}, 100),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Statistics
	api.HandleFunc("/stats", s.handleGetStats).Methods("GET")

	// Positions
	api.HandleFunc("/positions", s.handleGetPositions).Methods("GET")
	api.HandleFunc("/positions/{id}", s.handleGetPosition).Methods("GET")
	api.HandleFunc("/positions/open", s.handleGetOpenPositions).Methods("GET")

	// Orders
	api.HandleFunc("/orders/position/{id}", s.handleGetOrdersByPosition).Methods("GET")

	// Signals
	api.HandleFunc("/signals", s.handleGetSignals).Methods("GET")

	// Channels
	api.HandleFunc("/channels", s.handleGetChannels).Methods("GET")

	// Configuration
	api.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	api.HandleFunc("/config", s.handleUpdateConfig).Methods("PUT")

	// Binance Accounts
	api.HandleFunc("/accounts", s.handleGetAccounts).Methods("GET")
	api.HandleFunc("/accounts", s.handleCreateAccount).Methods("POST")
	api.HandleFunc("/accounts/{id}", s.handleGetAccount).Methods("GET")
	api.HandleFunc("/accounts/{id}", s.handleUpdateAccount).Methods("PUT")
	api.HandleFunc("/accounts/{id}", s.handleDeleteAccount).Methods("DELETE")
	api.HandleFunc("/accounts/{id}/set-default", s.handleSetDefaultAccount).Methods("POST")

	// WebSocket
	api.HandleFunc("/ws", s.handleWebSocket)

	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Serve static files (Vue.js app)
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/dist")))
}

// Start starts the web server
func (s *Server) Start() error {
	if !s.config.WebAPI.Enabled {
		s.logger.Info("Web API is disabled")
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.WebAPI.Host, s.config.WebAPI.Port)

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   s.config.WebAPI.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(s.router)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start WebSocket broadcaster
	go s.runWebSocketBroadcaster()

	s.logger.Infof("Starting web API server on %s", addr)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start web server: %w", err)
	}

	return nil
}

// Stop stops the web server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// Handler functions

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.repo.GetTradingStats()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get statistics")
		return
	}

	s.respondJSON(w, http.StatusOK, stats)
}

func (s *Server) handleGetPositions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	positions, err := s.repo.GetAllPositions(limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get positions")
		return
	}

	s.respondJSON(w, http.StatusOK, positions)
}

func (s *Server) handleGetPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid position ID")
		return
	}

	position, err := s.repo.GetPosition(id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get position")
		return
	}

	if position == nil {
		s.respondError(w, http.StatusNotFound, "Position not found")
		return
	}

	s.respondJSON(w, http.StatusOK, position)
}

func (s *Server) handleGetOpenPositions(w http.ResponseWriter, r *http.Request) {
	positions, err := s.repo.GetOpenPositions()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get open positions")
		return
	}

	s.respondJSON(w, http.StatusOK, positions)
}

func (s *Server) handleGetOrdersByPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid position ID")
		return
	}

	orders, err := s.repo.GetOrdersByPosition(id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get orders")
		return
	}

	s.respondJSON(w, http.StatusOK, orders)
}

func (s *Server) handleGetSignals(w http.ResponseWriter, r *http.Request) {
	// This would require a new repository method
	// For now, return empty array
	s.respondJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handleGetChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.repo.GetAllChannels()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get channels")
		return
	}

	s.respondJSON(w, http.StatusOK, channels)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Return safe config (without sensitive data)
	safeConfig := map[string]interface{}{
		"trading": map[string]interface{}{
			"enabled":          s.config.Trading.Enabled,
			"leverage":         s.config.Trading.Leverage,
			"order_amount":     s.config.Trading.OrderAmount,
			"target_percent":   s.config.Trading.TargetPercent,
			"stoploss_percent": s.config.Trading.StopLossPercent,
			"order_timeout":    s.config.Trading.OrderTimeout,
			"max_positions":    s.config.Trading.MaxPositions,
			"dry_run":          s.config.Trading.DryRun,
			"signal_pattern":   s.config.Trading.SignalPattern,
		},
	}

	s.respondJSON(w, http.StatusOK, safeConfig)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update config (this is simplified - in production, you'd want more validation)
	// Note: This only updates in-memory config, not the file
	if trading, ok := updates["trading"].(map[string]interface{}); ok {
		if v, ok := trading["enabled"].(bool); ok {
			s.config.Trading.Enabled = v
		}
		if v, ok := trading["leverage"].(float64); ok {
			s.config.Trading.Leverage = int(v)
		}
		if v, ok := trading["order_amount"].(float64); ok {
			s.config.Trading.OrderAmount = v
		}
		if v, ok := trading["target_percent"].(float64); ok {
			s.config.Trading.TargetPercent = v
		}
		if v, ok := trading["stoploss_percent"].(float64); ok {
			s.config.Trading.StopLossPercent = v
		}
		if v, ok := trading["dry_run"].(bool); ok {
			s.config.Trading.DryRun = v
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Binance Account handlers

func (s *Server) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.repo.GetAllAccounts()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get accounts")
		return
	}

	// Mask API secrets in response
	for _, acc := range accounts {
		if len(acc.APISecret) > 4 {
			acc.APISecret = acc.APISecret[:4] + "..." + acc.APISecret[len(acc.APISecret)-4:]
		}
	}

	s.respondJSON(w, http.StatusOK, accounts)
}

func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	account, err := s.repo.GetAccount(id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get account")
		return
	}

	if account == nil {
		s.respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Mask API secret
	if len(account.APISecret) > 4 {
		account.APISecret = account.APISecret[:4] + "..." + account.APISecret[len(account.APISecret)-4:]
	}

	s.respondJSON(w, http.StatusOK, account)
}

func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var account models.BinanceAccount
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if account.Name == "" || account.APIKey == "" || account.APISecret == "" {
		s.respondError(w, http.StatusBadRequest, "Name, API key, and API secret are required")
		return
	}

	// Set defaults
	account.IsActive = true
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	if err := s.repo.SaveAccount(&account); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	s.respondJSON(w, http.StatusCreated, account)
}

func (s *Server) handleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	var account models.BinanceAccount
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	account.ID = id
	account.UpdatedAt = time.Now()

	if err := s.repo.UpdateAccount(&account); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to update account")
		return
	}

	s.respondJSON(w, http.StatusOK, account)
}

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	if err := s.repo.DeleteAccount(id); err != nil {
		if err.Error() == "cannot delete account with open positions" {
			s.respondError(w, http.StatusBadRequest, err.Error())
		} else {
			s.respondError(w, http.StatusInternalServerError, "Failed to delete account")
		}
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleSetDefaultAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	account, err := s.repo.GetAccount(id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to get account")
		return
	}

	if account == nil {
		s.respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	account.IsDefault = true
	if err := s.repo.UpdateAccount(account); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to set default account")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "default account updated"})
}

// WebSocket handler
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	s.wsClientsMu.Lock()
	s.wsClients[conn] = true
	s.wsClientsMu.Unlock()

	s.logger.Info("New WebSocket client connected")

	// Send initial data
	stats, _ := s.repo.GetTradingStats()
	positions, _ := s.repo.GetOpenPositions()

	initialData := map[string]interface{}{
		"type":      "initial",
		"stats":     stats,
		"positions": positions,
	}

	conn.WriteJSON(initialData)

	// Handle client disconnection
	defer func() {
		s.wsClientsMu.Lock()
		delete(s.wsClients, conn)
		s.wsClientsMu.Unlock()
		conn.Close()
		s.logger.Info("WebSocket client disconnected")
	}()

	// Read messages from client (ping/pong)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// runWebSocketBroadcaster broadcasts messages to all WebSocket clients
func (s *Server) runWebSocketBroadcaster() {
	for message := range s.wsBroadcast {
		s.wsClientsMu.RLock()
		for client := range s.wsClients {
			err := client.WriteJSON(message)
			if err != nil {
				s.logger.Errorf("WebSocket write error: %v", err)
				client.Close()
				delete(s.wsClients, client)
			}
		}
		s.wsClientsMu.RUnlock()
	}
}

// BroadcastUpdate broadcasts an update to all WebSocket clients
func (s *Server) BroadcastUpdate(updateType string, data interface{}) {
	message := map[string]interface{}{
		"type": updateType,
		"data": data,
	}

	select {
	case s.wsBroadcast <- message:
	default:
		s.logger.Warn("WebSocket broadcast channel full, dropping message")
	}
}

// BroadcastPositionUpdate broadcasts a position update
func (s *Server) BroadcastPositionUpdate(position *models.Position) {
	s.BroadcastUpdate("position_update", position)
}

// BroadcastOrderUpdate broadcasts an order update
func (s *Server) BroadcastOrderUpdate(order *models.Order) {
	s.BroadcastUpdate("order_update", order)
}

// Helper functions

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}
