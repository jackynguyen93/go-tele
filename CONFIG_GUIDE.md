# Configuration Guide

## Understanding config.yaml with QR Code Authentication

Since the application now uses **QR code authentication**, the configuration has changed slightly.

## config.yaml Fields

### Telegram Section

```yaml
telegram:
  api_id: 12345678                    # Required - From my.telegram.org
  api_hash: "your_api_hash_here"      # Required - From my.telegram.org
  phone_number: "+1234567890"          # OPTIONAL - Not used with QR auth
  bot_token: ""                        # Optional - For bot authentication
  use_test_dc: false                   # Use test data center
```

**Important Changes:**
- `phone_number` is now **OPTIONAL** for user accounts (QR code auth doesn't use it)
- `phone_number` can be left empty: `phone_number: ""`
- Only `api_id` and `api_hash` are required for QR authentication

### Database Section

```yaml
database:
  type: "sqlite"                       # Database type
  dsn: "data/messages.db"             # Database file path
```

### TDLib Section

```yaml
tdlib:
  database_directory: "data/tdlib"    # TDLib data (authentication stored here)
  files_directory: "data/files"       # Downloaded files
  use_file_database: true
  use_chat_info_database: true
  use_message_database: true
  system_language: "en"
  device_model: "Server"
  system_version: "Linux"
  app_version: "1.0.0"
```

### Channels Section

```yaml
channels:
  - "telegram"                        # Channel username (without @)
  - "durov"
```

### Logging Section

```yaml
logging:
  level: "info"                       # debug, info, warn, error
  format: "text"                      # text or json
```

---

## How to Change Configuration

### Option 1: Edit on Host Machine (Recommended)

Since config.yaml is mounted as a volume, you can edit it directly:

```bash
# Edit the file
vim config.yaml

# Restart container to apply changes
docker compose restart
```

### Option 2: Using Environment Variables

You can override config values using environment variables in docker-compose.yml:

```yaml
services:
  tdclient:
    image: tdclient:cached
    environment:
      - TELEGRAM_API_ID=12345678
      - TELEGRAM_API_HASH=your_api_hash
      - LOG_LEVEL=debug
    # ... rest of config
```

---

## Re-authenticating with Different Account

To switch to a different Telegram account:

```bash
# 1. Stop the container
docker compose down

# 2. Remove authentication data
rm -rf data/tdlib/*

# 3. Start fresh - you'll get a new QR code
docker compose run --rm tdclient

# 4. Scan the QR code with the new account you want to use
```

**Note:** With QR code authentication, you don't need to change config.yaml to switch accounts. Just clear the authentication data and scan a new QR code with the account you want.

---

## Minimal config.yaml for QR Authentication

```yaml
telegram:
  api_id: YOUR_API_ID
  api_hash: "YOUR_API_HASH"
  phone_number: ""  # Can be empty for QR auth
  use_test_dc: false

database:
  type: "sqlite"
  dsn: "data/messages.db"

tdlib:
  database_directory: "data/tdlib"
  files_directory: "data/files"
  use_file_database: true
  use_chat_info_database: true
  use_message_database: true
  system_language: "en"
  device_model: "Server"
  system_version: "Linux"
  app_version: "1.0.0"

channels:
  - "telegram"

logging:
  level: "info"
  format: "text"
```

---

## Configuration in Docker

### Method 1: Volume Mount (Current Setup)

The config file is mounted from host:

```yaml
volumes:
  - ./config.yaml:/app/config.yaml:ro  # :ro means read-only
```

To edit:
```bash
# Edit on host
vim config.yaml

# Restart container
docker compose restart
```

### Method 2: Environment Variables

Override specific values:

```yaml
services:
  tdclient:
    environment:
      - TELEGRAM_API_ID=12345678
      - TELEGRAM_API_HASH=abc123
      - LOG_LEVEL=debug
```

### Method 3: Docker Secrets (Production)

For sensitive values in production:

```bash
# Create secrets
echo "your_api_hash" | docker secret create telegram_api_hash -

# Use in compose
services:
  tdclient:
    secrets:
      - telegram_api_hash
```

### Method 4: .env File

Create a `.env` file:

```bash
# .env
TELEGRAM_API_ID=12345678
TELEGRAM_API_HASH=your_api_hash
LOG_LEVEL=info
```

Reference in docker-compose.yml:

```yaml
services:
  tdclient:
    environment:
      - TELEGRAM_API_ID=${TELEGRAM_API_ID}
      - TELEGRAM_API_HASH=${TELEGRAM_API_HASH}
      - LOG_LEVEL=${LOG_LEVEL:-info}
```

---

## Editing Config Inside Container

If you need to edit config inside the running container:

```bash
# Enter the container
docker compose exec tdclient sh

# Edit config (if not read-only)
vi /app/config.yaml

# Exit
exit

# Restart to apply changes
docker compose restart
```

**Note:** Changes inside the container won't persist if the container is removed, unless you remove the `:ro` flag from the volume mount.

---

## Checking Current Configuration

```bash
# View mounted config
docker compose exec tdclient cat /app/config.yaml

# Check environment variables
docker compose exec tdclient env | grep TELEGRAM
```

---

## Common Configuration Changes

### Change Log Level

```bash
# Edit config.yaml
vim config.yaml
# Change: level: "debug"

# Or use environment variable
docker compose run -e LOG_LEVEL=debug --rm tdclient
```

### Add/Remove Channels

```bash
# Edit config.yaml
vim config.yaml
# Add to channels list

# Restart
docker compose restart
```

### Change Database Location

```bash
# Edit config.yaml
vim config.yaml
# Change: dsn: "data/messages.db"

# Make sure to update volume mount if needed
```

---

## Security Best Practices

1. **Never commit config.yaml to git**
   ```bash
   echo "config.yaml" >> .gitignore
   ```

2. **Use environment variables for secrets in production**
   ```yaml
   environment:
     - TELEGRAM_API_HASH=${TELEGRAM_API_HASH}
   ```

3. **Use Docker secrets for sensitive data**
   ```bash
   docker secret create telegram_api_hash -
   ```

4. **Set proper file permissions**
   ```bash
   chmod 600 config.yaml
   ```

5. **Use read-only mounts when possible**
   ```yaml
   volumes:
     - ./config.yaml:/app/config.yaml:ro
   ```
