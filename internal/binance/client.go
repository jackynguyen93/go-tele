package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// REST API endpoints
	DefaultBaseURL = "https://fapi.binance.com"
	TestnetBaseURL = "https://testnet.binancefuture.com"

	// WebSocket endpoints
	DefaultWSBaseURL = "wss://fstream.binance.com"
	TestnetWSBaseURL = "wss://stream.binancefuture.com"
)

// Client represents a Binance Futures API client
type Client struct {
	apiKey     string
	apiSecret  string
	baseURL    string
	wsBaseURL  string
	httpClient *http.Client
	logger     *logrus.Logger

	// WebSocket
	wsConn *websocket.Conn
	wsMu   sync.RWMutex

	// Callbacks
	onOrderUpdate    func(*OrderUpdate)
	onAccountUpdate  func(*AccountUpdate)
	onPositionUpdate func(*PositionUpdate)
}

// NewClient creates a new Binance Futures client
func NewClient(apiKey, apiSecret string, isTestnet bool, logger *logrus.Logger) *Client {
	baseURL := DefaultBaseURL
	wsBaseURL := DefaultWSBaseURL

	if isTestnet {
		baseURL = TestnetBaseURL
		wsBaseURL = TestnetWSBaseURL
	}

	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		wsBaseURL: wsBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// NewClientWithConfig creates a new Binance Futures client with custom URLs
func NewClientWithConfig(apiKey, apiSecret string, baseURL, wsBaseURL string, logger *logrus.Logger) *Client {
	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		wsBaseURL: wsBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// sign creates a signature for authenticated requests
func (c *Client) sign(params url.Values) string {
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	mac.Write([]byte(params.Encode()))
	return hex.EncodeToString(mac.Sum(nil))
}

// doRequest performs an HTTP request to Binance API
func (c *Client) doRequest(method, endpoint string, params url.Values, signed bool) ([]byte, error) {
	if signed {
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		params.Set("signature", c.sign(params))
	}

	reqURL := c.baseURL + endpoint
	if method == http.MethodGet || method == http.MethodDelete {
		if len(params) > 0 {
			reqURL += "?" + params.Encode()
		}
	}

	var reqBody io.Reader
	if method == http.MethodPost && len(params) > 0 {
		reqBody = strings.NewReader(params.Encode())
	}

	req, err := http.NewRequest(method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("binance API error [%d]: %s", apiErr.Code, apiErr.Msg)
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetExchangeInfo retrieves exchange trading rules and symbol information
func (c *Client) GetExchangeInfo() (*ExchangeInfo, error) {
	body, err := c.doRequest(http.MethodGet, "/fapi/v1/exchangeInfo", url.Values{}, false)
	if err != nil {
		return nil, err
	}

	var info ExchangeInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exchange info: %w", err)
	}

	return &info, nil
}

// GetSymbolPriceTicker gets the latest price for a symbol
func (c *Client) GetSymbolPriceTicker(symbol string) (*PriceTicker, error) {
	params := url.Values{}
	params.Set("symbol", symbol)

	body, err := c.doRequest(http.MethodGet, "/fapi/v1/ticker/price", params, false)
	if err != nil {
		return nil, err
	}

	var ticker PriceTicker
	if err := json.Unmarshal(body, &ticker); err != nil {
		return nil, fmt.Errorf("failed to unmarshal price ticker: %w", err)
	}

	return &ticker, nil
}

// SetLeverage changes the leverage for a symbol
func (c *Client) SetLeverage(symbol string, leverage int) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("leverage", strconv.Itoa(leverage))

	_, err := c.doRequest(http.MethodPost, "/fapi/v1/leverage", params, true)
	if err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	c.logger.Infof("Set leverage to %dx for %s", leverage, symbol)
	return nil
}

// SetMarginType sets the margin type for a symbol (ISOLATED or CROSSED)
func (c *Client) SetMarginType(symbol string, marginType string) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("marginType", marginType)

	_, err := c.doRequest(http.MethodPost, "/fapi/v1/marginType", params, true)
	if err != nil {
		// Margin type already set is not an error
		if strings.Contains(err.Error(), "No need to change margin type") {
			return nil
		}
		return fmt.Errorf("failed to set margin type: %w", err)
	}

	c.logger.Infof("Set margin type to %s for %s", marginType, symbol)
	return nil
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(order *NewOrder) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", order.Symbol)
	params.Set("side", order.Side)
	params.Set("type", order.Type)

	if order.Quantity > 0 {
		params.Set("quantity", fmt.Sprintf("%.8f", order.Quantity))
	}

	if order.Price > 0 {
		params.Set("price", fmt.Sprintf("%.8f", order.Price))
	}

	if order.StopPrice > 0 {
		params.Set("stopPrice", fmt.Sprintf("%.8f", order.StopPrice))
	}

	if order.TimeInForce != "" {
		params.Set("timeInForce", order.TimeInForce)
	}

	if order.ReduceOnly {
		params.Set("reduceOnly", "true")
	}

	if order.NewClientOrderID != "" {
		params.Set("newClientOrderId", order.NewClientOrderID)
	}

	body, err := c.doRequest(http.MethodPost, "/fapi/v1/order", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	var resp OrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"symbol":   resp.Symbol,
		"order_id": resp.OrderID,
		"side":     resp.Side,
		"type":     resp.Type,
		"status":   resp.Status,
	}).Info("Order placed successfully")

	return &resp, nil
}

// CancelOrder cancels an active order
func (c *Client) CancelOrder(symbol string, orderID int64) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", strconv.FormatInt(orderID, 10))

	body, err := c.doRequest(http.MethodDelete, "/fapi/v1/order", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	var resp OrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cancel response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"order_id": orderID,
	}).Info("Order canceled successfully")

	return &resp, nil
}

// QueryOrder checks an order's status
func (c *Client) QueryOrder(symbol string, orderID int64) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", strconv.FormatInt(orderID, 10))

	body, err := c.doRequest(http.MethodGet, "/fapi/v1/order", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to query order: %w", err)
	}

	var resp OrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order response: %w", err)
	}

	return &resp, nil
}

// GetAccount retrieves current account information
func (c *Client) GetAccount() (*AccountInfo, error) {
	params := url.Values{}

	body, err := c.doRequest(http.MethodGet, "/fapi/v2/account", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	var account AccountInfo
	if err := json.Unmarshal(body, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account info: %w", err)
	}

	return &account, nil
}

// GetPositions retrieves current positions
func (c *Client) GetPositions() ([]PositionRisk, error) {
	params := url.Values{}

	body, err := c.doRequest(http.MethodGet, "/fapi/v2/positionRisk", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var positions []PositionRisk
	if err := json.Unmarshal(body, &positions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal positions: %w", err)
	}

	return positions, nil
}

// StartUserDataStream starts a user data stream for WebSocket
func (c *Client) StartUserDataStream() (string, error) {
	params := url.Values{}

	body, err := c.doRequest(http.MethodPost, "/fapi/v1/listenKey", params, true)
	if err != nil {
		return "", fmt.Errorf("failed to start user data stream: %w", err)
	}

	var resp struct {
		ListenKey string `json:"listenKey"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal listen key: %w", err)
	}

	c.logger.Info("User data stream started")
	return resp.ListenKey, nil
}

// KeepAliveUserDataStream extends the validity of a user data stream
func (c *Client) KeepAliveUserDataStream(listenKey string) error {
	params := url.Values{}
	params.Set("listenKey", listenKey)

	_, err := c.doRequest(http.MethodPut, "/fapi/v1/listenKey", params, true)
	if err != nil {
		return fmt.Errorf("failed to keep alive user data stream: %w", err)
	}

	return nil
}

// ConnectUserDataStream connects to user data stream WebSocket
func (c *Client) ConnectUserDataStream(listenKey string) error {
	wsURL := fmt.Sprintf("%s/ws/%s", c.wsBaseURL, listenKey)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.wsMu.Lock()
	c.wsConn = conn
	c.wsMu.Unlock()

	c.logger.Info("Connected to user data stream WebSocket")

	// Start reading messages
	go c.readWebSocket()

	// Start keep-alive ticker
	go c.keepAliveStream(listenKey)

	return nil
}

// readWebSocket reads messages from the WebSocket connection
func (c *Client) readWebSocket() {
	defer func() {
		c.wsMu.Lock()
		if c.wsConn != nil {
			c.wsConn.Close()
			c.wsConn = nil
		}
		c.wsMu.Unlock()
	}()

	for {
		c.wsMu.RLock()
		conn := c.wsConn
		c.wsMu.RUnlock()

		if conn == nil {
			break
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.logger.Errorf("WebSocket read error: %v", err)
			break
		}

		c.handleWebSocketMessage(message)
	}
}

// handleWebSocketMessage processes WebSocket messages
func (c *Client) handleWebSocketMessage(message []byte) {
	var baseMsg struct {
		EventType string `json:"e"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		// Silently ignore unmarshal errors (likely heartbeat or unknown message types)
		return
	}

	switch baseMsg.EventType {
	case "ORDER_TRADE_UPDATE":
		var update OrderUpdate
		if err := json.Unmarshal(message, &update); err != nil {
			c.logger.Errorf("Failed to unmarshal order update: %v", err)
			return
		}
		if c.onOrderUpdate != nil {
			c.onOrderUpdate(&update)
		}

	case "ACCOUNT_UPDATE":
		var update AccountUpdate
		if err := json.Unmarshal(message, &update); err != nil {
			c.logger.Errorf("Failed to unmarshal account update: %v", err)
			return
		}
		if c.onAccountUpdate != nil {
			c.onAccountUpdate(&update)
		}

		// Extract position updates
		for _, pos := range update.UpdateData.Positions {
			if c.onPositionUpdate != nil {
				c.onPositionUpdate(&pos)
			}
		}
	}
}

// keepAliveStream keeps the listen key alive
func (c *Client) keepAliveStream(listenKey string) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := c.KeepAliveUserDataStream(listenKey); err != nil {
			c.logger.Errorf("Failed to keep alive user data stream: %v", err)
			return
		}
		c.logger.Debug("User data stream keep-alive sent")
	}
}

// SetOrderUpdateCallback sets the callback for order updates
func (c *Client) SetOrderUpdateCallback(callback func(*OrderUpdate)) {
	c.onOrderUpdate = callback
}

// SetAccountUpdateCallback sets the callback for account updates
func (c *Client) SetAccountUpdateCallback(callback func(*AccountUpdate)) {
	c.onAccountUpdate = callback
}

// SetPositionUpdateCallback sets the callback for position updates
func (c *Client) SetPositionUpdateCallback(callback func(*PositionUpdate)) {
	c.onPositionUpdate = callback
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()

	if c.wsConn != nil {
		err := c.wsConn.Close()
		c.wsConn = nil
		return err
	}

	return nil
}
