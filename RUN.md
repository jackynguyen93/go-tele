# Running the Telegram Channel Monitor

## First Time Setup (QR Code Authentication)

The application uses **QR code authentication** for user accounts. On first run:
1. The application will display a QR code link in the terminal
2. Open the link in your browser or scan it directly with your Telegram app
3. Go to Telegram app → Settings → Devices → Link Desktop Device
4. Scan the QR code displayed
5. Authentication completes automatically after scanning

## Run with Docker Compose (Interactive)

```bash
docker compose run --rm tdclient
```

This command:
- Runs the container interactively
- Connects your terminal to the container's stdin/stdout
- Allows you to see prompts and enter authentication details
- Removes the container when you exit (--rm)

## What You'll See

```
INFO[2025-11-12 08:44:43] Starting Telegram Channel Monitor...
INFO[2025-11-12 08:44:43] Database initialized successfully
[ 3][t 0][1762937083.802691221][Client.cpp:482]     Created managed client 1

INFO[2025-11-12 08:44:45] ========================================
INFO[2025-11-12 08:44:45] QR Code Authentication
INFO[2025-11-12 08:44:45] ========================================
INFO[2025-11-12 08:44:45] Please scan this QR code with your Telegram app:
INFO[2025-11-12 08:44:45]
INFO[2025-11-12 08:44:45] Link: tg://login?token=abcd1234...
INFO[2025-11-12 08:44:45]
INFO[2025-11-12 08:44:45] Or open this link in your browser:
INFO[2025-11-12 08:44:45] tg://login?token=abcd1234...
INFO[2025-11-12 08:44:45] ========================================

INFO[2025-11-12 08:44:50] ✓ Authorization successful!
INFO[2025-11-12 08:44:50] Telegram client started successfully
```

## After Authentication

Once authenticated, the session is saved in `./data/tdlib/` directory. Future runs won't require re-authentication:

```bash
docker compose up
```

## Troubleshooting

### QR code not displaying
If you don't see the QR code link, ensure you're running with `docker compose run --rm tdclient` (not `docker compose up`).

### Permission denied on data directory
```bash
mkdir -p data/tdlib data/files
chmod 755 data
```

### Already authenticated but want to re-authenticate
```bash
# Remove the session data
rm -rf data/tdlib/*
# Run interactive authentication again
docker compose run --rm tdclient
```

## Running in Background (After Authentication)

Once authenticated, you can run in detached mode:

```bash
docker compose up -d
```

View logs:
```bash
docker compose logs -f
```

Stop:
```bash
docker compose down
```

## Debug Mode

If you encounter authentication issues:

1. Check if you're already authenticated:
```bash
ls -la data/tdlib/
```

If you see files there, you're already authenticated. To re-authenticate:
```bash
rm -rf data/tdlib/*
docker compose run --rm tdclient
```

2. Run with debug logging:
Edit `config.yaml` and set:
```yaml
logging:
  level: "debug"
```

3. Check QR code link:
The QR code link format should be: `tg://login?token=...`
Copy this link and open it in your browser or paste it into Telegram's "Link Desktop Device" dialog.
