# Architecture Documentation

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         User / CLI                          │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    CLI Handler (cli.go)                     │
│  - Command processing                                       │
│  - User interaction                                         │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Channel Monitor (monitor.go)               │
│  - Channel subscription management                          │
│  - Message routing                                          │
│  - Handler coordination                                     │
└────────┬────────────────────────────────────┬───────────────┘
         │                                    │
         ▼                                    ▼
┌─────────────────────┐            ┌──────────────────────────┐
│  TDLib Client       │            │  Storage Repository      │
│  (client.go)        │            │  (repository.go)         │
│  - Authentication   │            │  - Database operations   │
│  - Message listener │            │  - Message persistence   │
│  - API calls        │            │  - Channel storage       │
└─────────┬───────────┘            └────────┬─────────────────┘
          │                                 │
          ▼                                 ▼
┌─────────────────────┐            ┌──────────────────────────┐
│  Telegram API       │            │  SQLite Database         │
│  (TDLib)            │            │  - messages table        │
│                     │            │  - channels table        │
└─────────────────────┘            └──────────────────────────┘
```

## Component Breakdown

### 1. Main Application (`cmd/tdclient/main.go`)

**Purpose**: Application entry point and lifecycle management

**Responsibilities**:
- Parse command-line flags
- Load configuration
- Initialize logger
- Create database connection
- Instantiate Telegram client
- Set up signal handlers for graceful shutdown
- Coordinate component startup and shutdown

**Key Functions**:
- `main()`: Entry point
- `setupLogger()`: Configure logging
- `createDirectories()`: Ensure required directories exist
- `reconnect()`: Handle reconnection logic

**Dependencies**:
- All internal packages
- Configuration system
- Database repository
- Telegram client

---

### 2. Configuration (`internal/config/config.go`)

**Purpose**: Configuration management and validation

**Data Structures**:
```go
Config
├── Telegram (API credentials, auth method)
├── Database (connection settings)
├── TDLib (TDLib parameters)
├── Channels (list of channels to monitor)
└── Logging (logging configuration)
```

**Key Functions**:
- `Load(path string)`: Read and parse YAML config
- `Validate()`: Ensure configuration is valid
- `IsBot()`: Check authentication mode

**Validation Rules**:
- API ID and Hash are required
- Either phone number or bot token must be provided
- Database DSN must be specified
- TDLib directories must be configured

---

### 3. TDLib Client (`internal/telegram/client.go`)

**Purpose**: Wrapper around TDLib with high-level API

**Core Functionality**:

#### Authentication
- Phone number authentication with 2FA support
- Bot token authentication
- Session persistence (handled by TDLib)
- Authorization state monitoring

#### Message Processing
- Real-time message listener
- Message type detection
- Content extraction
- Sender identification

#### Chat Operations
- Search for public chats
- Join channels
- Retrieve chat history
- Get chat information

**Key Types**:
```go
Client {
    tdClient  *client.Client    // TDLib client
    config    *config.Config    // Configuration
    logger    *logrus.Logger    // Logger
    handlers  []MessageHandler  // Message handlers
    ctx       context.Context   // Context for cancellation
    connected bool              // Connection status
}
```

**Key Functions**:
- `NewClient()`: Create and initialize client
- `Start()`: Authenticate and connect
- `StartListening()`: Listen for updates
- `GetChat()`: Get chat by username
- `JoinChat()`: Subscribe to a channel
- `GetChatHistory()`: Fetch historical messages
- `handleNewMessage()`: Process incoming messages
- `convertMessage()`: Convert TDLib message to internal model

**Message Handler Pattern**:
```go
type MessageHandler func(msg *models.Message) error
```

Handlers are called concurrently for each message, allowing parallel processing.

---

### 4. Channel Monitor (`internal/telegram/monitor.go`)

**Purpose**: High-level channel management and message coordination

**Responsibilities**:
- Maintain list of monitored channels
- Subscribe/unsubscribe from channels
- Route messages to storage
- Coordinate message handlers
- Fetch historical messages

**Key Types**:
```go
Monitor {
    client     *Client              // Telegram client
    repo       *Repository          // Database repository
    config     *Config              // Configuration
    logger     *Logger              // Logger
    channels   map[int64]*Channel   // Active channels
    channelsMu sync.RWMutex         // Thread-safe map access
}
```

**Key Functions**:
- `Start()`: Initialize monitoring
- `SubscribeChannel()`: Add channel subscription
- `UnsubscribeChannel()`: Remove channel subscription
- `FetchHistory()`: Retrieve historical messages
- `ListChannels()`: Get all monitored channels
- `handleMessage()`: Process and store messages

**Concurrency Model**:
- RWMutex for thread-safe channel map access
- Goroutines for parallel message processing
- Non-blocking message handlers

---

### 5. Storage Repository (`internal/storage/repository.go`)

**Purpose**: Database abstraction layer

**Design Pattern**: Repository Pattern
- Abstracts database operations
- Provides interface for easy testing
- Allows database engine swapping

**Database Operations**:

#### Messages
- `SaveMessage()`: Insert new message (INSERT OR IGNORE)
- `GetMessagesByChannel()`: Retrieve messages from channel

#### Channels
- `SaveChannel()`: Insert or update channel
- `GetChannel()`: Retrieve channel by ID or username
- `GetAllChannels()`: Get all active channels
- `DeactivateChannel()`: Mark channel as inactive

**Key Features**:
- WAL mode for concurrent reads/writes
- Automatic schema migration
- Inline schema fallback if migration file missing
- Prepared statements for safety
- Transaction support

**Schema**:
```sql
messages (
    id, message_id, channel_id, channel_name,
    sender_id, sender_name, text, media_type,
    is_forwarded, timestamp, created_at
)

channels (
    id, channel_id, username, title,
    is_active, created_at, updated_at
)
```

**Indexes**:
- `idx_channels_channel_id`: Fast channel lookups
- `idx_channels_username`: Username searches
- `idx_messages_channel_id`: Messages by channel
- `idx_messages_timestamp`: Time-based queries
- `idx_messages_sender_id`: Sender lookups

---

### 6. CLI Interface (`internal/cli/cli.go`)

**Purpose**: Interactive command-line interface

**Features**:
- Command parsing and validation
- Real-time user interaction
- Channel management commands
- Status monitoring
- Help system

**Commands**:
| Command | Function | Example |
|---------|----------|---------|
| `help` | Show help | `help` |
| `list`, `ls` | List channels | `list` |
| `add`, `subscribe` | Add channel | `add telegram` |
| `remove`, `unsubscribe` | Remove channel | `remove 12345` |
| `history`, `fetch` | Fetch history | `history 12345 50` |
| `status` | Show status | `status` |
| `quit`, `exit` | Exit app | `quit` |

**Implementation**:
- `bufio.Scanner` for input reading
- Command parser with argument validation
- Formatted output with box drawing
- Interactive prompts for confirmations

---

### 7. Data Models (`pkg/models/message.go`)

**Purpose**: Define core data structures

**Models**:

```go
Message {
    ID          int64      // Database ID
    MessageID   int64      // Telegram message ID
    ChannelID   int64      // Channel ID
    ChannelName string     // Channel name
    SenderID    int64      // Sender ID
    SenderName  string     // Sender name
    Text        string     // Message text
    MediaType   string     // Message type
    IsForwarded bool       // Forwarded flag
    Timestamp   time.Time  // Message time
    CreatedAt   time.Time  // Storage time
}

Channel {
    ID        int64      // Database ID
    ChannelID int64      // Telegram ID
    Username  string     // @username
    Title     string     // Display name
    IsActive  bool       // Active flag
    CreatedAt time.Time  // Created time
    UpdatedAt time.Time  // Updated time
}
```

---

## Data Flow

### 1. Startup Sequence

```
main()
  → Load config
  → Initialize logger
  → Create database
  → Initialize TDLib client
  → Authenticate
  → Create monitor
  → Subscribe to channels
  → Start message listener
  → Start CLI
  → Wait for shutdown signal
```

### 2. Message Reception Flow

```
Telegram API
  → TDLib Update
  → Client.handleNewMessage()
  → Client.convertMessage()
  → Monitor.handleMessage()
  → Repository.SaveMessage()
  → Database
```

### 3. User Command Flow

```
User Input
  → CLI.handleCommand()
  → Monitor.SubscribeChannel()
  → Client.JoinChat()
  → Repository.SaveChannel()
  → Database
```

### 4. Graceful Shutdown Flow

```
Signal (SIGINT/SIGTERM)
  → Cancel context
  → Stop message listener
  → Close TDLib client
  → Close database
  → Exit
```

---

## Concurrency Model

### Thread Safety

**RWMutex Usage**:
- `Monitor.channelsMu`: Protects channels map
- Read locks for lookups
- Write locks for modifications

**Goroutines**:
- Message listener runs in dedicated goroutine
- CLI runs in dedicated goroutine
- Each message handler runs concurrently
- All goroutines respect context cancellation

**Channel Communication**:
- TDLib updates via listener channel
- Signal handling via os.Signal channel
- Error propagation via error channels

### Synchronization Points

1. **Message Processing**: Handlers called in goroutines
2. **Database Writes**: Serialized by SQLite
3. **Channel Map Access**: Protected by RWMutex
4. **Shutdown**: Context cancellation coordinates all goroutines

---

## Error Handling

### Strategy

1. **Log and Continue**: Non-critical errors (e.g., failed to parse message)
2. **Retry**: Transient errors (e.g., network issues)
3. **Fail Fast**: Configuration errors, missing dependencies
4. **Graceful Degradation**: Continue with partial functionality

### Error Propagation

- Errors wrapped with context using `fmt.Errorf()`
- Structured logging with fields
- Critical errors logged at ERROR level
- Non-critical at WARN level

---

## Performance Considerations

### Optimizations

1. **Database**: WAL mode for concurrent access
2. **Indexes**: Proper indexing for common queries
3. **Goroutines**: Parallel message processing
4. **Buffering**: Channel buffers for message queues
5. **Connection Pooling**: Database connection reuse

### Bottlenecks

1. **Database Writes**: Serial by nature in SQLite
2. **TDLib Updates**: Single listener thread
3. **Message Handlers**: Limited by database write speed

### Scalability

- **Horizontal**: Multiple instances with load balancer
- **Vertical**: More CPU/RAM for more channels
- **Database**: Migrate to PostgreSQL for higher throughput

---

## Extension Points

### Adding New Message Handlers

```go
monitor.client.AddMessageHandler(func(msg *models.Message) error {
    // Your custom logic
    return nil
})
```

### Supporting New Databases

Implement these methods:
```go
type Repository interface {
    SaveMessage(msg *models.Message) error
    GetMessagesByChannel(channelID int64, limit int) ([]*models.Message, error)
    SaveChannel(channel *models.Channel) error
    GetChannel(identifier string) (*models.Channel, error)
    GetAllChannels() ([]*models.Channel, error)
    DeactivateChannel(channelID int64) error
    Close() error
}
```

### Adding New CLI Commands

1. Add case to `handleCommand()` in `cli.go`
2. Implement command handler function
3. Update help text

### Adding New Configuration Options

1. Add field to appropriate config struct
2. Update validation logic
3. Update config.yaml.example
4. Use in relevant component

---

## Testing Strategy

### Unit Tests
- Mock TDLib client
- Mock database repository
- Test business logic in isolation

### Integration Tests
- Test with real SQLite database
- Mock Telegram API
- Test component interaction

### End-to-End Tests
- Use test Telegram data center
- Test full user flows
- Verify data persistence

---

## Security Architecture

### Threat Model

**Protected Against**:
- SQL injection (prepared statements)
- Unauthorized access (authentication required)
- Data loss (graceful shutdown)
- Credential exposure (gitignore)

**Not Protected Against**:
- Physical access to server
- Database file theft (no encryption)
- Compromised Telegram account

### Best Practices

1. ✅ Credentials not in code
2. ✅ Config file not in version control
3. ✅ Prepared statements for queries
4. ✅ Input validation
5. ✅ Proper file permissions
6. ⚠️ Consider database encryption for sensitive data

---

## Deployment Architecture

### Single Server
```
[Server]
  ├── tdclient binary
  ├── config.yaml
  ├── data/
  │   ├── messages.db
  │   ├── tdlib/
  │   └── files/
  └── systemd service
```

### High Availability (Future)
```
[Load Balancer]
      │
      ├── [Server 1] ─┐
      ├── [Server 2] ─┼── [PostgreSQL Cluster]
      └── [Server 3] ─┘
```

---

## Monitoring & Observability

### Logging
- Structured logs with logrus
- Configurable log levels
- JSON format option for log aggregation

### Metrics (Future)
- Message throughput
- Error rates
- Channel count
- Database size
- API latency

### Alerting (Future)
- Connection failures
- Authentication issues
- Database errors
- Disk space warnings

---

## Maintenance

### Regular Tasks
1. Monitor disk usage
2. Backup database
3. Update dependencies
4. Review logs
5. Check error rates

### Updates
1. Update TDLib when available
2. Update Go dependencies (`go get -u`)
3. Test in development first
4. Backup before production update

---

This architecture supports the current requirements and provides clear extension points for future enhancements.
