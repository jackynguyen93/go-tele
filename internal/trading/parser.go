package trading

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"tdlib-go/internal/config"
	"tdlib-go/pkg/models"
)

// SignalParser parses trading signals from Telegram messages
type SignalParser struct {
	pattern *regexp.Regexp
	logger  *logrus.Logger
}

// NewSignalParser creates a new signal parser
func NewSignalParser(cfg *config.Config, logger *logrus.Logger) (*SignalParser, error) {
	pattern, err := regexp.Compile(cfg.Trading.SignalPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid signal pattern: %w", err)
	}

	return &SignalParser{
		pattern: pattern,
		logger:  logger,
	}, nil
}

// Parse attempts to parse a trading signal from a message
func (p *SignalParser) Parse(msg *models.Message) (*models.Signal, error) {
	if msg.Text == "" {
		return nil, nil
	}

	// Try to match the pattern
	matches := p.pattern.FindStringSubmatch(msg.Text)
	if len(matches) == 0 {
		return nil, nil
	}

	// Extract symbol from the first capturing group
	var symbol string
	if len(matches) > 1 {
		symbol = strings.ToUpper(strings.TrimSpace(matches[1]))
	} else {
		return nil, fmt.Errorf("no symbol captured from pattern")
	}

	// Normalize symbol for Binance Futures (ensure it ends with USDT)
	symbol = p.normalizeSymbol(symbol)

	p.logger.WithFields(logrus.Fields{
		"channel_id": msg.ChannelID,
		"message_id": msg.MessageID,
		"symbol":     symbol,
	}).Info("Trading signal detected")

	signal := &models.Signal{
		MessageID:  msg.MessageID,
		ChannelID:  msg.ChannelID,
		Symbol:     symbol,
		RawMessage: msg.Text,
		ParsedAt:   time.Now(),
		Status:     "pending",
	}

	return signal, nil
}

// normalizeSymbol normalizes a symbol for Binance Futures
func (p *SignalParser) normalizeSymbol(symbol string) string {
	// Remove common prefixes/suffixes
	symbol = strings.TrimPrefix(symbol, "$")
	symbol = strings.TrimPrefix(symbol, "#")
	symbol = strings.TrimSuffix(symbol, "/USDT")
	symbol = strings.TrimSuffix(symbol, "-USDT")
	symbol = strings.TrimSuffix(symbol, "_USDT")

	// Ensure USDT suffix for futures
	if !strings.HasSuffix(symbol, "USDT") {
		symbol = symbol + "USDT"
	}

	return symbol
}

// IsValidSymbol checks if a symbol is valid for trading
func (p *SignalParser) IsValidSymbol(symbol string) bool {
	// Basic validation
	if len(symbol) < 4 || len(symbol) > 20 {
		return false
	}

	// Must end with USDT for futures
	if !strings.HasSuffix(symbol, "USDT") {
		return false
	}

	// Must be alphanumeric
	matched, _ := regexp.MatchString(`^[A-Z0-9]+$`, symbol)
	return matched
}

// ExtractMultipleSymbols extracts all matching symbols from a message
func (p *SignalParser) ExtractMultipleSymbols(text string) []string {
	matches := p.pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	symbols := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			symbol := p.normalizeSymbol(strings.ToUpper(strings.TrimSpace(match[1])))
			if p.IsValidSymbol(symbol) && !seen[symbol] {
				symbols = append(symbols, symbol)
				seen[symbol] = true
			}
		}
	}

	return symbols
}
