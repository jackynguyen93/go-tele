# Deployment Guide

This guide explains how to push your Docker image to a registry and deploy it on a server.

## Option 1: Docker Hub (Public/Private Registry)

### Step 1: Tag the Image

```bash
# Tag your image with your Docker Hub username
docker tag tdclient:cached YOUR_DOCKERHUB_USERNAME/tdclient:v0.7.6
docker tag tdclient:cached YOUR_DOCKERHUB_USERNAME/tdclient:latest
```

### Step 2: Login to Docker Hub

```bash
docker login
# Enter your Docker Hub username and password
```

### Step 3: Push to Docker Hub

```bash
docker push YOUR_DOCKERHUB_USERNAME/tdclient:v0.7.6
docker push YOUR_DOCKERHUB_USERNAME/tdclient:latest
```

### Step 4: On Your Server

```bash
# Pull the image
docker pull YOUR_DOCKERHUB_USERNAME/tdclient:latest

# Run with docker compose
# First, create docker-compose.prod.yml on your server
```

Create `docker-compose.prod.yml` on your server:
```yaml
services:
  tdclient:
    image: YOUR_DOCKERHUB_USERNAME/tdclient:latest
    container_name: telegram-channel-monitor
    restart: unless-stopped

    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/app/data

    stdin_open: true
    tty: true

    environment:
      - LOG_LEVEL=${LOG_LEVEL:-info}

    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

Then run:
```bash
# First time (for QR code authentication)
docker compose -f docker-compose.prod.yml run --rm tdclient

# After authentication
docker compose -f docker-compose.prod.yml up -d
```

---

## Option 2: GitHub Container Registry (ghcr.io)

### Step 1: Create Personal Access Token

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token with `write:packages` and `read:packages` permissions
3. Save the token

### Step 2: Login to GitHub Container Registry

```bash
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

### Step 3: Tag and Push

```bash
# Tag the image
docker tag tdclient:cached ghcr.io/YOUR_GITHUB_USERNAME/tdclient:v0.7.6
docker tag tdclient:cached ghcr.io/YOUR_GITHUB_USERNAME/tdclient:latest

# Push to GitHub Container Registry
docker push ghcr.io/YOUR_GITHUB_USERNAME/tdclient:v0.7.6
docker push ghcr.io/YOUR_GITHUB_USERNAME/tdclient:latest
```

### Step 4: On Your Server

```bash
# Login on server
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin

# Pull the image
docker pull ghcr.io/YOUR_GITHUB_USERNAME/tdclient:latest

# Update docker-compose.prod.yml to use ghcr.io image
```

Update image in `docker-compose.prod.yml`:
```yaml
services:
  tdclient:
    image: ghcr.io/YOUR_GITHUB_USERNAME/tdclient:latest
    # ... rest of config
```

---

## Option 3: Private Registry (Self-Hosted)

### Step 1: Set Up Private Registry (on server)

```bash
docker run -d -p 5000:5000 --restart=always --name registry registry:2
```

### Step 2: Tag and Push (from local machine)

```bash
# Tag the image
docker tag tdclient:cached YOUR_SERVER_IP:5000/tdclient:latest

# Push to private registry
docker push YOUR_SERVER_IP:5000/tdclient:latest
```

### Step 3: On Your Server

Update `docker-compose.prod.yml`:
```yaml
services:
  tdclient:
    image: localhost:5000/tdclient:latest
    # ... rest of config
```

---

## Option 4: Save/Load (No Registry)

If you don't want to use a registry, you can transfer the image directly:

### Step 1: Save Image to File

```bash
docker save tdclient:cached | gzip > tdclient-cached.tar.gz
```

### Step 2: Transfer to Server

```bash
# Using scp
scp tdclient-cached.tar.gz user@your-server:/path/to/destination/

# Or using rsync
rsync -avz tdclient-cached.tar.gz user@your-server:/path/to/destination/
```

### Step 3: Load on Server

```bash
# On the server
gunzip -c tdclient-cached.tar.gz | docker load

# Verify image is loaded
docker images | grep tdclient
```

### Step 4: Update docker-compose.prod.yml

```yaml
services:
  tdclient:
    image: tdclient:cached
    # ... rest of config
```

---

## Server Setup

### 1. Transfer Configuration Files

```bash
# From local machine
scp config.yaml user@your-server:/path/to/app/
scp docker-compose.prod.yml user@your-server:/path/to/app/docker-compose.yml
```

### 2. Create Data Directories

```bash
# On the server
mkdir -p data/tdlib data/files
chmod 755 data
```

### 3. First Run (Authentication)

```bash
# Interactive run for QR code authentication
docker compose run --rm tdclient
```

When you see the QR code link:
1. Copy the `tg://login?token=...` link
2. Open it in your browser or paste into Telegram
3. Scan the QR code with your Telegram app
4. Wait for "Authorization successful"

### 4. Run in Background

```bash
# After authentication
docker compose up -d

# Check logs
docker compose logs -f

# Check status
docker compose ps
```

---

## Production Best Practices

### 1. Use Docker Compose Override

Create `docker-compose.override.yml` for local development:
```yaml
services:
  tdclient:
    build:
      context: .
      dockerfile: Dockerfile.cached
```

This way `docker-compose.yml` uses the registry image in production, but local development can still build from source.

### 2. Version Tagging

Always tag your images with version numbers:
```bash
docker tag tdclient:cached YOUR_REGISTRY/tdclient:v0.7.6
docker tag tdclient:cached YOUR_REGISTRY/tdclient:latest
```

### 3. Health Checks

Add health check to docker-compose.yml:
```yaml
services:
  tdclient:
    # ... other config
    healthcheck:
      test: ["CMD", "pgrep", "-f", "tdclient"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### 4. Resource Limits

```yaml
services:
  tdclient:
    # ... other config
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### 5. Automated Backups

```bash
# Backup script (backup.sh)
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
tar -czf backup_${DATE}.tar.gz data/
# Upload to S3, rsync to backup server, etc.
```

---

## Example: Complete Deployment to Server

```bash
# 1. On local machine - Build and push
docker compose build
docker tag tdclient:cached myusername/tdclient:v0.7.6
docker push myusername/tdclient:v0.7.6

# 2. On server - Setup
ssh user@server
mkdir -p ~/telegram-monitor
cd ~/telegram-monitor

# 3. Create config.yaml on server
cat > config.yaml << 'EOF'
telegram:
  api_id: YOUR_API_ID
  api_hash: "YOUR_API_HASH"
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
EOF

# 4. Create docker-compose.yml on server
cat > docker-compose.yml << 'EOF'
services:
  tdclient:
    image: myusername/tdclient:v0.7.6
    container_name: telegram-channel-monitor
    restart: unless-stopped

    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/app/data

    stdin_open: true
    tty: true

    environment:
      - LOG_LEVEL=info

    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
EOF

# 5. Setup and authenticate
mkdir -p data/tdlib data/files
docker compose run --rm tdclient  # Scan QR code here

# 6. Run in background
docker compose up -d

# 7. Monitor
docker compose logs -f
```

---

## Troubleshooting

### Can't pull image
- Check you're logged in: `docker login`
- Check image name is correct
- For private registries, ensure credentials are set

### Permission issues on server
```bash
sudo chown -R $USER:$USER data/
chmod -R 755 data/
```

### Container won't start
```bash
# Check logs
docker compose logs tdclient

# Check if already authenticated
ls -la data/tdlib/
```

### Update to new version
```bash
# Pull new version
docker compose pull

# Recreate container
docker compose up -d --force-recreate
```
