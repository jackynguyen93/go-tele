# How to Find and Join Telegram Channels Without Username

Some Telegram channels don't have a public username (like `@channel_name`). Here's how to find and join them.

## Method 1: Use Invite Link

### If you have an invite link like:
- `https://t.me/+AbCdEfGhIjKlMnO`
- `https://t.me/joinchat/AbCdEfGhIjKlMnO`

### Step 1: Join the channel manually first
1. Open the link in Telegram app
2. Click "Join"
3. The channel will appear in your chat list

### Step 2: Get the channel ID using the CLI
Once the application is running:
```
> add https://t.me/+AbCdEfGhIjKlMnO
```

Or you can search by the channel title if you know it.

---

## Method 2: Get Channel ID from Telegram Web

### Step 1: Open Telegram Web
Go to https://web.telegram.org/

### Step 2: Navigate to the channel
Click on the channel you want to monitor

### Step 3: Check the URL
The URL will look like:
```
https://web.telegram.org/k/#-1001234567890
```

The number after `#-` is the channel ID: `-1001234567890`

### Step 4: Use in config.yaml
You can now use the channel ID directly in your config:

```yaml
channels:
  - "-1001234567890"  # Channel ID
```

---

## Method 3: Use Telegram Desktop

### Step 1: Enable Debug Mode
- Close Telegram Desktop
- Right-click Telegram icon
- Add `--debug` to the shortcut target

### Step 2: View Channel ID
- Open Telegram Desktop
- Right-click on the channel
- Select "Copy Debug Info"
- The channel ID will be in the copied text

---

## Method 4: Get All Joined Channels

If you've already joined the channel in your Telegram app, you can list all your channels.

### Create a helper script to list channels:

```bash
# Run the application in interactive mode
docker compose run --rm tdclient

# In the CLI, type:
> list
```

This will show all channels you've joined with their IDs and usernames (if they have one).

---

## Method 5: Use @userinfobot

### Step 1: Join the channel
Join the channel manually in Telegram app

### Step 2: Forward a message
1. Forward any message from that channel to [@userinfobot](https://t.me/userinfobot)
2. The bot will reply with channel information including the channel ID

### Example Response:
```
Channel: Channel Name
ID: -1001234567890
```

---

## Method 6: Use Python Script (Advanced)

If you want to list all your channels programmatically:

```python
from telethon import TelegramClient

api_id = YOUR_API_ID
api_hash = 'YOUR_API_HASH'

client = TelegramClient('session', api_id, api_hash)

async def main():
    dialogs = await client.get_dialogs()
    for dialog in dialogs:
        if dialog.is_channel:
            print(f"Title: {dialog.title}")
            print(f"ID: {dialog.id}")
            print(f"Username: {dialog.entity.username if hasattr(dialog.entity, 'username') else 'No username'}")
            print("---")

with client:
    client.loop.run_until_complete(main())
```

---

## Supported Channel Identifier Formats

The application supports multiple formats:

### 1. Username (with or without @)
```yaml
channels:
  - "telegram"      # Without @
  - "@durov"        # With @
```

### 2. Channel ID (numeric)
```yaml
channels:
  - "-1001234567890"  # Full channel ID
```

### 3. Invite Link
```yaml
channels:
  - "https://t.me/+AbCdEfGhIjKlMnO"
  - "+AbCdEfGhIjKlMnO"  # Just the hash part
```

### 4. Public Link
```yaml
channels:
  - "https://t.me/telegram"
  - "telegram"  # Same as above
```

---

## Example config.yaml with Mixed Formats

```yaml
telegram:
  api_id: 12345678
  api_hash: "your_api_hash"
  phone_number: ""
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
  # Public channels with username
  - "telegram"
  - "@durov"

  # Private channels by ID
  - "-1001234567890"
  - "-1009876543210"

  # Channels by invite link
  - "+AbCdEfGhIjKlMnO"
  - "https://t.me/+XyZaBcDeFgHiJ"

logging:
  level: "info"
  format: "text"
```

---

## Troubleshooting

### "Channel not found" error

**Solution 1: Join the channel first**
```bash
# Manually join in Telegram app first, then add to config
```

**Solution 2: Use the correct format**
```yaml
# Wrong
channels:
  - "1234567890"  # Missing the -100 prefix

# Correct
channels:
  - "-1001234567890"
```

### "Access denied" error

Some channels are:
- Private (requires admin approval)
- Restricted to certain users
- Bot-restricted (doesn't allow bots)

**Solution:** Make sure:
1. You're using a user account (not bot) for QR authentication
2. You've joined the channel manually in Telegram app first
3. The channel allows automated access

### Can't get channel ID

**Quick method:**
1. Join the channel in Telegram
2. Run the application
3. Use the CLI `list` command to see all channels with their IDs

---

## Getting Channel ID Programmatically

You can also add a channel by joining it interactively and then checking its ID.

### Using the CLI:

```bash
# Start the application
docker compose run --rm tdclient

# In the CLI:
> add @channel_username
Joining channel...
✓ Successfully joined channel: Channel Name (ID: -1001234567890)

# Now you can use the ID in config.yaml
```

---

## Quick Reference

| Channel Type | Format | Example |
|-------------|--------|---------|
| Public with username | `username` or `@username` | `telegram`, `@durov` |
| Private/No username | `-100XXXXXXXXXX` | `-1001234567890` |
| Invite link (full) | `https://t.me/+HASH` | `https://t.me/+AbCdEfGh` |
| Invite link (hash) | `+HASH` | `+AbCdEfGh` |
| Public link | `https://t.me/username` | `https://t.me/telegram` |

---

## Recommended Workflow

1. **Join the channel** manually in your Telegram app first
2. **Get the channel ID** using one of the methods above
3. **Add to config.yaml** using the channel ID format
4. **Restart the application** to start monitoring

```bash
# Edit config
nano config.yaml

# Add the channel ID to the channels list
# channels:
#   - "-1001234567890"

# Restart
docker compose restart

# Verify it's monitoring
docker compose logs -f
```
