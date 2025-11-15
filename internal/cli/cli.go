package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"tdlib-go/internal/telegram"
)

// CLI provides a command-line interface for the application
type CLI struct {
	monitor *telegram.Monitor
	logger  *logrus.Logger
	scanner *bufio.Scanner
}

// NewCLI creates a new CLI instance
func NewCLI(monitor *telegram.Monitor, logger *logrus.Logger) *CLI {
	return &CLI{
		monitor: monitor,
		logger:  logger,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// Start starts the interactive CLI
func (c *CLI) Start() {
	c.printWelcome()
	c.printHelp()

	for {
		fmt.Print("\n> ")
		if !c.scanner.Scan() {
			break
		}

		line := strings.TrimSpace(c.scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		if err := c.handleCommand(command, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// handleCommand processes a CLI command
func (c *CLI) handleCommand(command string, args []string) error {
	switch command {
	case "help":
		c.printHelp()
	case "list", "ls":
		c.listChannels()
	case "add", "subscribe":
		return c.addChannel(args)
	case "remove", "unsubscribe":
		return c.removeChannel(args)
	case "history", "fetch":
		return c.fetchHistory(args)
	case "status":
		c.showStatus()
	case "quit", "exit":
		fmt.Println("Exiting...")
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Type 'help' for available commands")
	}

	return nil
}

// printWelcome prints the welcome message
func (c *CLI) printWelcome() {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║     Telegram Channel Monitor - TDLib Go Client       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
}

// printHelp prints available commands
func (c *CLI) printHelp() {
	fmt.Println("\nAvailable Commands:")
	fmt.Println("  help                          - Show this help message")
	fmt.Println("  list, ls                      - List all monitored channels")
	fmt.Println("  add <identifier>              - Add/subscribe to a channel")
	fmt.Println("                                  Supports: username, channel ID, or invite link")
	fmt.Println("  remove <channel_id>           - Remove/unsubscribe from a channel")
	fmt.Println("  history <channel_id> <limit>  - Fetch historical messages")
	fmt.Println("  status                        - Show connection status")
	fmt.Println("  quit, exit                    - Exit the application")
	fmt.Println("\nExamples:")
	fmt.Println("  add telegram                              (username)")
	fmt.Println("  add -1002233859472                        (channel ID)")
	fmt.Println("  add https://t.me/+wIr66-O-XaxjOWI0        (invite link)")
	fmt.Println("  remove 1234567890")
	fmt.Println("  history 1234567890 50")
}

// listChannels lists all monitored channels
func (c *CLI) listChannels() {
	channels := c.monitor.ListChannels()

	if len(channels) == 0 {
		fmt.Println("No channels are being monitored")
		return
	}

	fmt.Printf("\nMonitored Channels (%d):\n", len(channels))
	fmt.Println("─────────────────────────────────────────────────────────")

	for i, ch := range channels {
		status := "✓ Active"
		if !ch.IsActive {
			status = "✗ Inactive"
		}

		fmt.Printf("%d. %s\n", i+1, ch.Title)
		fmt.Printf("   ID: %d | Username: %s | Status: %s\n",
			ch.ChannelID, ch.Username, status)
		fmt.Printf("   Subscribed: %s\n", ch.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("─────────────────────────────────────────────────────────")
}

// addChannel subscribes to a new channel
func (c *CLI) addChannel(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: add <identifier> (username, channel ID, or invite link)")
	}

	identifier := args[0]

	// Remove @ prefix if present for usernames
	identifier = strings.TrimPrefix(identifier, "@")

	// Detect and display the type of identifier
	var identifierType string
	if strings.Contains(identifier, "t.me/") || strings.Contains(identifier, "telegram.me/") {
		identifierType = "invite link"
	} else if _, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		identifierType = "channel ID"
	} else {
		identifierType = "username"
	}

	fmt.Printf("Subscribing to channel by %s: %s...\n", identifierType, identifier)

	if err := c.monitor.SubscribeChannel(identifier); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	fmt.Printf("✓ Successfully subscribed!\n")

	// Ask if user wants to fetch history
	fmt.Print("Fetch message history? (y/N): ")
	if c.scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(c.scanner.Text()))
		if response == "y" || response == "yes" {
			fmt.Print("How many messages to fetch? (default: 50): ")
			if c.scanner.Scan() {
				limitStr := strings.TrimSpace(c.scanner.Text())
				limit := 50
				if limitStr != "" {
					if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
						limit = parsedLimit
					}
				}

				// Get the channel info to fetch history
				channels := c.monitor.ListChannels()
				var channelID int64

				// If identifier is a channel ID, use it directly
				if identifierType == "channel ID" {
					if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
						channelID = id
					}
				} else {
					// Otherwise, find the most recently added channel (last in list)
					// This works for both username and invite link cases
					if len(channels) > 0 {
						// Find the channel that was just added
						for _, ch := range channels {
							if ch.Username == identifier || ch.ChannelID > channelID {
								channelID = ch.ChannelID
							}
						}
					}
				}

				if channelID != 0 {
					fmt.Printf("Fetching %d messages...\n", limit)
					if err := c.monitor.FetchHistory(channelID, int32(limit)); err != nil {
						return fmt.Errorf("failed to fetch history: %w", err)
					}
					fmt.Println("✓ History fetched successfully")
				} else {
					fmt.Println("⚠ Could not determine channel ID for history fetch")
				}
			}
		}
	}

	return nil
}

// removeChannel unsubscribes from a channel
func (c *CLI) removeChannel(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: remove <channel_id>")
	}

	channelID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %w", err)
	}

	fmt.Printf("Unsubscribing from channel %d...\n", channelID)

	if err := c.monitor.UnsubscribeChannel(channelID); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	fmt.Printf("✓ Successfully unsubscribed from channel %d\n", channelID)

	return nil
}

// fetchHistory fetches historical messages from a channel
func (c *CLI) fetchHistory(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: history <channel_id> <limit>")
	}

	channelID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %w", err)
	}

	limit, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid limit: %w", err)
	}

	fmt.Printf("Fetching %d messages from channel %d...\n", limit, channelID)

	if err := c.monitor.FetchHistory(channelID, int32(limit)); err != nil {
		return fmt.Errorf("failed to fetch history: %w", err)
	}

	fmt.Println("✓ History fetched successfully")

	return nil
}

// showStatus shows the current connection status
func (c *CLI) showStatus() {
	channels := c.monitor.ListChannels()

	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                   System Status                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Printf("  Monitored Channels: %d\n", len(channels))
	fmt.Println("  Status: Running")
	fmt.Println("─────────────────────────────────────────────────────────")
}
