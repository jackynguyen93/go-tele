# Telegram-to-Binance Futures Trading Bot

A high-performance Telegram client built with TDLib and Go that monitors Telegram channels for trading signals and automatically executes Binance Futures orders.

## Features

### Core Trading
- 🚀 **Fast Order Execution**: Places entry, TP, and SL orders simultaneously (< 100ms)
- 📊 **Signal Parsing**: Flexible regex-based pattern matching for trading signals
- 💹 **Binance Futures**: Full integration with REST + WebSocket API
- 🎯 **Auto TP/SL**: Automatic take-profit and stop-loss calculation with leverage
- ⏱️ **Order Timeout**: Auto-cancel unfilled TP/SL orders after configurable timeout
- 📈 **Position Tracking**: Real-time PnL calculation and position management
- 🖥️ **Web Dashboard**: Vue.js dashboard with real-time WebSocket updates

### Telegram Monitoring
- 🔐 **Flexible Authentication**: Supports both user accounts and bot tokens
- 📡 **Real-time Monitoring**: Subscribe to multiple channels and receive messages instantly
- 💾 **Message Archiving**: Automatically stores all messages in SQLite database
- 📜 **Historical Messages**: Fetch and archive historical messages from channels
- 🔄 **Concurrent Processing**: Efficient message handling with goroutines
- 🛡️ **Graceful Shutdown**: Proper cleanup and reconnection handling
- 🖥️ **Interactive CLI**: Easy-to-use command-line interface

## Quick Start with Docker 🐳 (Recommended!)

**No TDLib installation needed!** Run with Docker in just a few steps:

```bash
# 1. Build TDLib base image (one-time, ~15 minutes)
./build-tdlib-base.sh

# 2. Configure your Telegram credentials
cp config.example.yaml config.yaml
# Edit config.yaml and add your Telegram API credentials
# (api_id, api_hash, phone_number)
# Note: Binance API keys are now managed via web dashboard!

# 3. Build the application (~30 seconds with cached TDLib base)
docker compose build

# 4. Run the application
docker compose up

# 5. Authenticate with Telegram (first run only)
# - A QR code link will appear in the terminal
# - Open the link and scan the QR code with your Telegram app
# - Authentication is saved in ./data directory

# Done! 🎉
# Access web dashboard at http://localhost:8080
```

### Docker Environment Management

```bash
# Run in detached mode (background)
docker compose up -d

# View logs
docker compose logs -f tdclient

# Stop the application
docker compose down

# Rebuild after code changes
docker compose build --no-cache

# Remove all data (WARNING: deletes database and auth)
rm -rf ./data
```

### Docker Data Persistence

The following directories are mounted from your host:
- `./config.yaml` - Configuration file (read-only)
- `./data/` - Persistent data:
  - `trading.db` - SQLite database (positions, orders, accounts)
  - `tdlib/` - Telegram authentication and cache
  - `files/` - Downloaded Telegram files

**Important**: Keep the `./data` directory to preserve your authentication and trading history.

### Setting Up Your First Trading Account

After starting the application with Docker:

1. **Complete Telegram Authentication** (first run only)
   - Scan the QR code shown in terminal with Telegram app
   - Authentication is saved automatically

2. **Access the Web Dashboard**
   - Open http://localhost:8080 in your browser

3. **Add Your First Binance Account**
   - Navigate to **Accounts** page (🔑 icon in sidebar)
   - Click **"+ Add Account"**
   - Enter account details:
     - **Account Name**: e.g., "Testnet Main" or "Production Account"
     - **API Key**: Your Binance API key
     - **API Secret**: Your Binance API secret
     - **Use Testnet**: ✅ Enable for testnet (recommended first!)
     - **Active**: ✅ Enable to use this account
     - **Set as Default**: ✅ Use this account for new trades
   - Click **"Add Account"**

4. **Configure Trading Settings**
   - Edit `config.yaml` to set:
     - Telegram channels to monitor
     - Trading parameters (leverage, position size, TP/SL percentages)
     - Signal pattern (regex to match trading signals)
   - Restart the container for config changes to take effect

5. **Start Trading!**
   - The bot will monitor your configured Telegram channels
   - When a signal is detected, it automatically executes orders
   - Monitor positions in real-time on the **Dashboard** and **Positions** pages

---

## Prerequisites (Native Build)

Before you begin with native build, ensure you have:

1. **Go 1.24 or higher** installed
2. **TDLib** (commit 971684a) installed on your system
3. **Telegram API credentials** (see below)

### Installing TDLib

This project uses [zelenin/go-tdlib v0.7.6](https://github.com/zelenin/go-tdlib) which requires **TDLib commit 971684a** (updated 2025-04-30) and supports **QR code authentication**.

#### Build from source (Recommended)
```bash
git clone https://github.com/tdlib/td.git
cd td
git checkout 971684a
mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local ..
make -j$(nproc)
sudo make install
```

#### macOS (Homebrew - may be older version)
```bash
brew install tdlib
# Note: Homebrew version may not match exactly. Build from source if issues occur.
```

#### Ubuntu/Debian (System packages - may be older version)
```bash
sudo apt-get update
sudo apt-get install -y libtdjson-dev
# Note: System packages may not match exactly. Build from source if issues occur.
```

### Getting API Credentials

#### Telegram API Credentials

1. Visit https://my.telegram.org
2. Log in with your phone number
3. Go to "API development tools"
4. Create a new application:
   - App title: Choose any name
   - Short name: Choose a short identifier
   - Platform: Other
5. Save your `api_id` and `api_hash`

#### Binance API Credentials

Binance API keys are now managed through the **web dashboard** at http://localhost:8080/accounts. You can add multiple accounts for different strategies or testing.

**For Testing (Recommended First)**:
1. Go to https://testnet.binancefuture.com
2. Login with GitHub/Google
3. Click "Generate HMAC_SHA256 Key"
4. Save the API Key and Secret
5. Get test funds from the faucet
6. Add the account via the web dashboard with "Testnet" option enabled

**For Production**:
1. Go to https://www.binance.com/en/my/settings/api-management
2. Create a new API key
3. Enable "Enable Futures" permission
4. **Enable IP whitelist for security**
5. Save the key and secret
6. Add the account via the web dashboard

## Installation

1. **Clone or navigate to the project directory:**
```bash
cd /Users/jacky/IdeaProjects/tdlib-go
```

2. **Install Go dependencies:**
```bash
go mod download
```

3. **Create configuration file:**
```bash
cp config.yaml.example config.yaml
```

4. **Edit `config.yaml` with your credentials:**
```yaml
telegram:
  api_id: 12345678  # Your API ID
  api_hash: "your_api_hash_here"  # Your API Hash
  phone_number: "+1234567890"  # Your phone number
```

## Configuration

### Configuration File (`config.yaml`)

```yaml
telegram:
  api_id: 12345678                    # From my.telegram.org
  api_hash: "your_api_hash_here"      # From my.telegram.org
  phone_number: "+1234567890"         # For user auth
  # bot_token: "your_bot_token"       # For bot auth (alternative)
  use_test_dc: false                  # Use test data center

database:
  type: "sqlite"
  dsn: "data/trading.db"              # Database file path

tdlib:
  database_directory: "data/tdlib"    # TDLib data
  files_directory: "data/files"       # Downloaded files
  use_file_database: true
  use_chat_info_database: true
  use_message_database: true
  system_language: "en"
  device_model: "Server"
  system_version: "Linux"
  app_version: "1.0.0"

channels:
  - "@trading_signals_channel"        # Channels to monitor for signals
  - "https://t.me/crypto_signals"

# Binance Futures Configuration (API keys are managed via web dashboard)
binance:
  base_url: ""                        # Optional: Custom REST API URL
  ws_base_url: ""                     # Optional: Custom WebSocket URL
  # Note: API credentials are stored in database and managed via web dashboard at /accounts

# Trading Configuration
trading:
  enabled: true                       # Enable/disable trading
  leverage: 10                        # Leverage for positions (1-125x)
  order_amount: 100                   # Position size in USDT
  target_percent: 0.02                # Take profit: 2%
  stoploss_percent: 0.01              # Stop loss: 1%
  order_timeout: 3600                 # Auto-cancel TP/SL after 3600 seconds
  signal_pattern: '(?i)\$([A-Z]{2,10})\b'  # Regex to match signals (e.g., $BTC)
  max_positions: 3                    # Maximum concurrent positions
  dry_run: false                      # If true, parse signals but don't trade

# Web API Configuration
webapi:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  cors_origins:
    - "http://localhost:3000"
    - "http://localhost:8080"

logging:
  level: "info"                       # debug, info, warn, error
  format: "text"                      # text or json
```

### Authentication Modes

**User Authentication** (Regular Account):
```yaml
phone_number: "+1234567890"
# bot_token: ""  # Leave empty or omit
```

**Bot Authentication**:
```yaml
phone_number: ""  # Leave empty or omit
bot_token: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
```

## Usage

### Building the Application

```bash
go build -o tdclient ./cmd/tdclient
```

### Running the Application

```bash
./tdclient -config config.yaml
```

Or with custom log level:
```bash
./tdclient -config config.yaml -log-level debug
```

### Accessing the Web Dashboard

Once the application is running, access the dashboard at:
- **Dashboard**: http://localhost:8080
- **Positions**: http://localhost:8080/positions
- **Accounts**: http://localhost:8080/accounts (Manage Binance API keys)
- **Settings**: http://localhost:8080/settings
- **API Stats**: http://localhost:8080/api/stats

The dashboard provides:
- Real-time trading statistics (win rate, total PnL)
- Active and closed positions with PnL tracking
- **Multiple Binance account management** (add/edit/delete accounts)
- Default account selection for new trades
- Live configuration updates
- WebSocket updates for real-time data

#### Managing Binance Accounts

Navigate to the **Accounts** page to:
- Add multiple Binance accounts (testnet or production)
- Edit API credentials
- Set a default account for trading
- Toggle accounts active/inactive
- View masked API keys for security
- Delete unused accounts (protected if they have open positions)

### First Run - Authentication

On first run, the application uses **QR code authentication** for user accounts:

1. **QR Code Authentication (User accounts):**
   - A QR code link will be displayed in the terminal
   - Open the link in your browser or scan it with your Telegram app
   - The QR code will appear on the screen
   - Scan it with your Telegram app (Settings → Devices → Link Desktop Device)
   - Authentication completes automatically after scanning

2. **Bot Authentication:**
   - Authentication is automatic using the bot token in config.yaml

### Interactive CLI Commands

Once running, you can use these commands:

| Command | Description | Example |
|---------|-------------|---------|
| `help` | Show available commands | `help` |
| `list` or `ls` | List all monitored channels | `list` |
| `add <username>` | Subscribe to a channel | `add telegram` |
| `remove <channel_id>` | Unsubscribe from a channel | `remove 1234567890` |
| `history <channel_id> <limit>` | Fetch historical messages | `history 1234567890 100` |
| `status` | Show connection status | `status` |
| `quit` or `exit` | Exit the application | `quit` |

### Example Session

```
> list
Monitored Channels (2):
─────────────────────────────────────────────────────────
1. Telegram
   ID: 1234567890 | Username: telegram | Status: ✓ Active
   Subscribed: 2024-01-15 10:30:00
2. Durov's Channel
   ID: 9876543210 | Username: durov | Status: ✓ Active
   Subscribed: 2024-01-15 10:31:00
─────────────────────────────────────────────────────────

> add techcrunch
Subscribing to channel: @techcrunch...
✓ Successfully subscribed to @techcrunch
Fetch message history? (y/N): y
How many messages to fetch? (default: 50): 100
Fetching 100 messages...
✓ History fetched successfully

> status
╔═══════════════════════════════════════════════════════╗
║                   System Status                       ║
╚═══════════════════════════════════════════════════════╝
  Monitored Channels: 3
  Status: Running
─────────────────────────────────────────────────────────
```

## Message Output

When a new message is received, it's displayed in this format:

```
[2024-01-15 14:30:45] User_123456789 (Channel: Telegram)
  Type: text | Forwarded: false
  Message: Check out our new features!
---
```

## Database Schema

### Trading Tables

- **binance_accounts**: Multiple Binance account credentials (API keys stored in DB)
- **signals**: Parsed trading signals from Telegram
- **positions**: Open and closed positions with PnL (linked to specific accounts)
- **orders**: All Binance orders (entry, TP, SL)
- **messages**: Archived Telegram messages
- **channels**: Monitored Telegram channels

### Key Features

- **Multiple accounts**: Support for multiple Binance accounts per installation
- **Account isolation**: Each position is linked to a specific Binance account
- **Default account**: One account can be set as default for new trades
- **Async logging**: Orders are logged to database asynchronously
- **Position tracking**: Automatic PnL calculation
- **Statistics**: Win rate, average win/loss, total PnL
- **WebSocket updates**: Real-time position and order updates

## Project Structure

```
tdlib-go/
├── cmd/
│   └── tdclient/          # Main application entry point
│       └── main.go
├── internal/
│   ├── binance/           # Binance Futures API client
│   │   ├── client.go      # REST + WebSocket client
│   │   └── types.go       # API models
│   ├── cli/               # Command-line interface
│   │   └── cli.go
│   ├── config/            # Configuration management
│   │   └── config.go
│   ├── storage/           # Database layer
│   │   └── repository.go
│   ├── telegram/          # Telegram client wrapper
│   │   ├── client.go      # TDLib client wrapper
│   │   └── monitor.go     # Channel monitoring logic
│   ├── trading/           # Trading engine
│   │   ├── engine.go      # Main trading engine
│   │   ├── executor.go    # Order execution
│   │   └── parser.go      # Signal parsing
│   └── webapi/            # Web API server
│       └── server.go      # REST + WebSocket API
├── pkg/
│   └── models/            # Data models
│       ├── message.go     # Telegram messages
│       └── trading.go     # Trading models
├── web/                   # Vue.js dashboard
│   ├── src/
│   │   ├── views/
│   │   │   ├── Dashboard.vue   # Trading statistics
│   │   │   ├── Positions.vue   # Active/closed positions
│   │   │   ├── Accounts.vue    # Binance account management
│   │   │   └── Settings.vue    # Configuration
│   │   ├── App.vue
│   │   ├── router.js
│   │   └── main.js
│   └── package.json
├── config.example.yaml    # Example configuration
├── docker-compose.yml     # Fast multi-phase build
├── Dockerfile.fast        # Uses pre-built TDLib base
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Development

### Running in Development Mode

```bash
go run ./cmd/tdclient -config config.yaml -log-level debug
```

### Building for Production

```bash
# Build optimized binary
go build -ldflags="-s -w" -o tdclient ./cmd/tdclient

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o tdclient-linux ./cmd/tdclient
GOOS=darwin GOARCH=arm64 go build -o tdclient-macos ./cmd/tdclient
GOOS=windows GOARCH=amd64 go build -o tdclient.exe ./cmd/tdclient
```

### Extending the Storage Layer

The storage layer is designed to be easily replaceable. To add support for PostgreSQL or MySQL:

1. Implement the repository interface in `internal/storage/`
2. Update the configuration to support the new database type
3. Modify `cmd/tdclient/main.go` to instantiate the correct repository

## Troubleshooting

### TDLib Not Found

**Error**: `cannot find -ltdjson`

**Solution**: Ensure TDLib is properly installed and in your library path:
```bash
export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH  # Linux
export DYLD_LIBRARY_PATH=/usr/local/lib:$DYLD_LIBRARY_PATH  # macOS
```

### Authentication Fails

**Issue**: Can't log in or receive "Invalid phone number"

**Solutions**:
- Ensure phone number is in international format: `+1234567890`
- Check that your API credentials are correct
- Try using `use_test_dc: true` for testing
- Verify your account is not restricted

### Permission Denied on Channel

**Issue**: Can't join or access a channel

**Solutions**:
- Ensure the channel is public or you have access
- Try joining the channel manually first in Telegram app
- Some channels may restrict bots

### Database Locked

**Issue**: `database is locked` error

**Solutions**:
- Ensure only one instance is running
- Check file permissions on the database directory
- WAL mode is enabled by default for better concurrency

## Performance Tips

1. **Concurrent Processing**: Message handlers run concurrently via goroutines
2. **WAL Mode**: SQLite runs in WAL mode for better write performance
3. **Batch Operations**: Consider batching database writes for high-volume channels
4. **Index Usage**: Database indexes are created for common queries
5. **Message Database**: Disable TDLib message database if not needed

## Security Considerations

- **API Credentials**:
  - Binance API keys are stored in the SQLite database (`data/trading.db`)
  - Never commit `config.yaml` or `data/` directory to version control
  - API secrets are masked in web dashboard responses (only show first/last 4 chars)
  - Consider encrypting the SQLite database for production use
- **Binance API Key Permissions**:
  - Enable **IP whitelist** on Binance for additional security
  - Only enable "Enable Futures" permission (no withdrawal permissions needed)
  - Use separate API keys for testnet and production
  - Rotate API keys periodically
- **2FA**: Enable two-factor authentication on your Telegram account
- **Bot Tokens**: Rotate bot tokens periodically if using bot authentication
- **File Permissions**: Ensure database and config files have proper permissions (600 or 640)
- **Docker Security**:
  - Config file is mounted as read-only in container
  - Data directory is isolated from container
  - Use docker secrets for production deployments

## Trading Bot Usage

### How It Works

1. **Signal Detection**: Bot monitors Telegram channels for messages matching your regex pattern
2. **Symbol Extraction**: Extracts token symbols (e.g., `$BTC` → `BTCUSDT`)
3. **Price Discovery**: Gets current market price from Binance
4. **Order Execution**: Places 3 orders simultaneously:
   - Entry: Market order (instant fill)
   - Take Profit: At calculated TP price
   - Stop Loss: At calculated SL price
5. **Position Tracking**: Monitors via WebSocket for order fills
6. **Auto-Cancel**: Cancels unfilled TP/SL after timeout

### Example Signal Flow

```
Telegram: "Buy $BTC now!"
    ↓
Parse: BTC → BTCUSDT
    ↓
Price: $50,000
    ↓
Calculate (with 10x leverage):
  - Entry: $50,000
  - TP: $60,000 (20% profit)
  - SL: $45,000 (10% loss)
    ↓
Execute 3 orders in parallel (< 100ms)
    ↓
Async: Log to database
    ↓
Dashboard: Updates in real-time
```

### Safety Features

- **Dry Run Mode**: Test signal parsing without real orders
- **Testnet Support**: Practice with fake money
- **Max Positions**: Limit concurrent exposure
- **Order Timeout**: Prevent stale orders
- **IP Whitelist**: Secure your Binance API key

### ⚠️ Important Warnings

- **Start with testnet**: Always test on https://testnet.binancefuture.com first
- **Use small positions**: Start with minimum order amounts
- **Understand leverage**: 10x leverage = 10x risk
- **Monitor actively**: Watch your first trades closely
- **Never risk more than you can afford to lose**

## Limitations

- **Private Channels**: Can only monitor channels you have access to
- **Rate Limits**: Respects Telegram's rate limiting
- **Message History**: Limited by Telegram's API (typically last 100 messages for channels)
- **Bot Restrictions**: Bots have more limited access compared to user accounts
- **Trading Risk**: Cryptocurrency trading involves substantial risk of loss

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

This project is provided as-is for educational and personal use.

## Disclaimer

**⚠️ TRADING DISCLAIMER ⚠️**

This software executes real trades with real money. Cryptocurrency trading involves substantial risk of loss.

- This software is provided "as-is" without any warranties
- The authors are NOT responsible for any financial losses
- Past performance does not guarantee future results
- Always start with testnet and small amounts
- Never invest more than you can afford to lose
- Understand the risks of leverage trading
- This is NOT financial advice

**USE AT YOUR OWN RISK**

## Acknowledgments

- [TDLib](https://github.com/tdlib/td) - Telegram Database Library (commit 971684a)
- [go-tdlib v0.7.6](https://github.com/zelenin/go-tdlib) - Go bindings for TDLib
- [logrus](https://github.com/sirupsen/logrus) - Structured logger for Go

## Support

For issues and questions:
- Check the [TDLib documentation](https://core.telegram.org/tdlib/docs/)
- Review the [Telegram API documentation](https://core.telegram.org/api)
- Open an issue in this repository

---

**Note**: This tool is for personal and educational use. Please respect Telegram's Terms of Service and use responsibly.
