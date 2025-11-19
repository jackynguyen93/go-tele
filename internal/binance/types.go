package binance

// APIError represents a Binance API error response
type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ExchangeInfo represents exchange trading rules and symbol information
type ExchangeInfo struct {
	Symbols []SymbolInfo `json:"symbols"`
}

// SymbolInfo represents information about a trading symbol
type SymbolInfo struct {
	Symbol                string  `json:"symbol"`
	Status                string  `json:"status"`
	BaseAsset             string  `json:"baseAsset"`
	QuoteAsset            string  `json:"quoteAsset"`
	PricePrecision        int     `json:"pricePrecision"`
	QuantityPrecision     int     `json:"quantityPrecision"`
	BaseAssetPrecision    int     `json:"baseAssetPrecision"`
	QuotePrecision        int     `json:"quotePrecision"`
}

// PriceTicker represents a price ticker
type PriceTicker struct {
	Symbol string  `json:"symbol"`
	Price  string  `json:"price"`
	Time   int64   `json:"time"`
}

// NewOrder represents a new order request
type NewOrder struct {
	Symbol           string
	Side             string  // BUY or SELL
	Type             string  // MARKET, LIMIT, STOP_MARKET, TAKE_PROFIT_MARKET
	Quantity         float64
	Price            float64
	StopPrice        float64
	TimeInForce      string  // GTC, IOC, FOK
	ReduceOnly       bool
	NewClientOrderID string
}

// OrderResponse represents an order response from Binance
type OrderResponse struct {
	OrderID       int64   `json:"orderId"`
	Symbol        string  `json:"symbol"`
	Status        string  `json:"status"`
	ClientOrderID string  `json:"clientOrderId"`
	Price         string  `json:"price"`
	AvgPrice      string  `json:"avgPrice"`
	OrigQty       string  `json:"origQty"`
	ExecutedQty   string  `json:"executedQty"`
	CumQty        string  `json:"cumQty"`
	CumQuote      string  `json:"cumQuote"`
	TimeInForce   string  `json:"timeInForce"`
	Type          string  `json:"type"`
	ReduceOnly    bool    `json:"reduceOnly"`
	Side          string  `json:"side"`
	StopPrice     string  `json:"stopPrice"`
	WorkingType   string  `json:"workingType"`
	UpdateTime    int64   `json:"updateTime"`
}

// AccountInfo represents account information
type AccountInfo struct {
	Assets                      []Asset    `json:"assets"`
	Positions                   []Position `json:"positions"`
	TotalInitialMargin          string     `json:"totalInitialMargin"`
	TotalMaintMargin            string     `json:"totalMaintMargin"`
	TotalWalletBalance          string     `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string     `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string     `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string     `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string     `json:"totalOpenOrderInitialMargin"`
	AvailableBalance            string     `json:"availableBalance"`
	MaxWithdrawAmount           string     `json:"maxWithdrawAmount"`
	CanTrade                    bool       `json:"canTrade"`
	CanDeposit                  bool       `json:"canDeposit"`
	CanWithdraw                 bool       `json:"canWithdraw"`
	UpdateTime                  int64      `json:"updateTime"`
}

// Asset represents an asset balance
type Asset struct {
	Asset                  string `json:"asset"`
	WalletBalance          string `json:"walletBalance"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	MarginBalance          string `json:"marginBalance"`
	MaintMargin            string `json:"maintMargin"`
	InitialMargin          string `json:"initialMargin"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
	CrossWalletBalance     string `json:"crossWalletBalance"`
	CrossUnPnl             string `json:"crossUnPnl"`
	AvailableBalance       string `json:"availableBalance"`
}

// Position represents a position
type Position struct {
	Symbol                 string `json:"symbol"`
	InitialMargin          string `json:"initialMargin"`
	MaintMargin            string `json:"maintMargin"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	Leverage               string `json:"leverage"`
	Isolated               bool   `json:"isolated"`
	EntryPrice             string `json:"entryPrice"`
	MaxNotional            string `json:"maxNotional"`
	PositionSide           string `json:"positionSide"`
	PositionAmt            string `json:"positionAmt"`
	Notional               string `json:"notional"`
}

// PositionRisk represents position risk information
type PositionRisk struct {
	Symbol           string `json:"symbol"`
	PositionAmt      string `json:"positionAmt"`
	EntryPrice       string `json:"entryPrice"`
	MarkPrice        string `json:"markPrice"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	LiquidationPrice string `json:"liquidationPrice"`
	Leverage         string `json:"leverage"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	MarginType       string `json:"marginType"`
	IsolatedMargin   string `json:"isolatedMargin"`
	IsAutoAddMargin  string `json:"isAutoAddMargin"`
	PositionSide     string `json:"positionSide"`
	Notional         string `json:"notional"`
	IsolatedWallet   string `json:"isolatedWallet"`
	UpdateTime       int64  `json:"updateTime"`
}

// OrderUpdate represents an order update from WebSocket
type OrderUpdate struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	Order     struct {
		Symbol        string `json:"s"`
		ClientOrderID string `json:"c"`
		Side          string `json:"S"`
		Type          string `json:"o"`
		TimeInForce   string `json:"f"`
		OrigQty       string `json:"q"`
		Price         string `json:"p"`
		AvgPrice      string `json:"ap"`
		StopPrice     string `json:"sp"`
		ExecutionType string `json:"x"`
		OrderStatus   string `json:"X"`
		OrderID       int64  `json:"i"`
		LastFilledQty string `json:"l"`
		FilledQty     string `json:"z"`
		LastFilledPrice string `json:"L"`
		CommissionAsset string `json:"N"`
		Commission      string `json:"n"`
		OrderTradeTime  int64  `json:"T"`
		TradeID         int64  `json:"t"`
		RealizedProfit  string `json:"rp"`
		ReduceOnly      bool   `json:"R"`
		WorkingType     string `json:"wt"`
		OrigType        string `json:"ot"`
		PositionSide    string `json:"ps"`
		IsCloseAll      bool   `json:"cp"`
		ActivationPrice string `json:"AP"`
		CallbackRate    string `json:"cr"`
	} `json:"o"`
}

// AccountUpdate represents an account update from WebSocket
type AccountUpdate struct {
	EventType  string `json:"e"`
	EventTime  int64  `json:"E"`
	UpdateData struct {
		Reason    string            `json:"m"`
		Balances  []BalanceUpdate   `json:"B"`
		Positions []PositionUpdate  `json:"P"`
	} `json:"a"`
}

// BalanceUpdate represents a balance update
type BalanceUpdate struct {
	Asset              string `json:"a"`
	WalletBalance      string `json:"wb"`
	CrossWalletBalance string `json:"cw"`
	BalanceChange      string `json:"bc"`
}

// PositionUpdate represents a position update
type PositionUpdate struct {
	Symbol                    string `json:"s"`
	PositionAmount            string `json:"pa"`
	EntryPrice                string `json:"ep"`
	AccumulatedRealized       string `json:"cr"`
	UnrealizedPnL             string `json:"up"`
	MarginType                string `json:"mt"`
	IsolatedWallet            string `json:"iw"`
	PositionSide              string `json:"ps"`
}
