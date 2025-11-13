# Telegram Channel Monitor

A high-performance Telegram client built with TDLib and Go that monitors and archives messages from Telegram channels in real-time.

## Features

- 🔐 **Flexible Authentication**: Supports both user accounts and bot tokens
- 📡 **Real-time Monitoring**: Subscribe to multiple channels and receive messages instantly
- 💾 **Message Archiving**: Automatically stores all messages in SQLite database
- 📜 **Historical Messages**: Fetch and archive historical messages from channels
- 🔄 **Concurrent Processing**: Efficient message handling with goroutines
- 🛡️ **Graceful Shutdown**: Proper cleanup and reconnection handling
- 🖥️ **Interactive CLI**: Easy-to-use command-line interface
- 📊 **Rich Message Support**: Handles text, media, forwarded messages, and more
- 🔌 **Extensible Storage**: Database layer designed to be easily swappable

## Quick Start with Docker 🐳 (Recommended!)

**No TDLib installation needed!** Run with Docker in 3 steps:

```bash
# 1. Configure (edit with your credentials)
cp config.yaml.example config.yaml

# 2. Build
docker compose build

# 3. Run
docker compose up

# Done! 🎉
```

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

### Getting Telegram API Credentials

1. Visit https://my.telegram.org
2. Log in with your phone number
3. Go to "API development tools"
4. Create a new application:
   - App title: Choose any name
   - Short name: Choose a short identifier
   - Platform: Other
5. Save your `api_id` and `api_hash`

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
  phone_number: "+1234567890"          # For user auth
  # bot_token: "your_bot_token"       # For bot auth (alternative)
  use_test_dc: false                   # Use test data center

database:
  type: "sqlite"
  dsn: "data/messages.db"             # Database file path

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
  - "telegram"                        # Channels to monitor
  - "durov"

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

### Messages Table

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Auto-increment primary key |
| message_id | INTEGER | Telegram message ID |
| channel_id | INTEGER | Telegram channel ID |
| channel_name | TEXT | Channel display name |
| sender_id | INTEGER | Sender's user ID |
| sender_name | TEXT | Sender's name |
| text | TEXT | Message text content |
| media_type | TEXT | Type of media (text, photo, video, etc.) |
| is_forwarded | BOOLEAN | Whether message is forwarded |
| timestamp | TIMESTAMP | When message was sent |
| created_at | TIMESTAMP | When record was created |

### Channels Table

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Auto-increment primary key |
| channel_id | INTEGER | Telegram channel ID |
| username | TEXT | Channel username |
| title | TEXT | Channel display name |
| is_active | BOOLEAN | Monitoring status |
| created_at | TIMESTAMP | When subscription was created |
| updated_at | TIMESTAMP | Last update time |

## Project Structure

```
tdlib-go/
├── cmd/
│   └── tdclient/          # Main application entry point
│       └── main.go
├── internal/
│   ├── cli/               # Command-line interface
│   │   └── cli.go
│   ├── config/            # Configuration management
│   │   └── config.go
│   ├── storage/           # Database layer
│   │   └── repository.go
│   └── telegram/          # Telegram client wrapper
│       ├── client.go      # TDLib client wrapper
│       └── monitor.go     # Channel monitoring logic
├── pkg/
│   └── models/            # Data models
│       └── message.go
├── migrations/            # Database migrations
│   └── 001_initial_schema.sql
├── config.yaml.example    # Example configuration
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

- **API Credentials**: Never commit `config.yaml` to version control
- **Database Encryption**: Consider encrypting the SQLite database for sensitive data
- **2FA**: Enable two-factor authentication on your Telegram account
- **Bot Tokens**: Rotate bot tokens periodically
- **File Permissions**: Ensure database and config files have proper permissions

## Limitations

- **Private Channels**: Can only monitor channels you have access to
- **Rate Limits**: Respects Telegram's rate limiting
- **Message History**: Limited by Telegram's API (typically last 100 messages for channels)
- **Bot Restrictions**: Bots have more limited access compared to user accounts

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

This project is provided as-is for educational and personal use.

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
