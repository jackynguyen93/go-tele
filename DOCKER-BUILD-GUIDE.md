# Docker Build Guide - Fast Rebuilds

This guide explains how to avoid rebuilding TDLib every time you change your code.

## Problem
The default `Dockerfile.cached` rebuilds TDLib from source (~10 minutes) even when only your Go code changes.

## Solution
Use a two-step approach:
1. Build TDLib base image **once** (one-time 10-minute setup)
2. Use fast rebuilds that only compile your Go code (~30 seconds)

---

## Quick Start

### Step 1: Build TDLib Base Image (ONE-TIME ONLY)
```bash
./build-tdlib-base.sh
```
Or manually:
```bash
docker build -f Dockerfile.tdlib -t tdlib-base:1.8.19 .
```

This builds TDLib and saves it as `tdlib-base:1.8.19` image.
**You only run this ONCE** (or when upgrading TDLib version).

### Step 2: Fast Rebuilds (Use This Every Time)
```bash
docker compose -f docker-compose.fast.yml build
```

This only rebuilds your Go application code in ~30 seconds.

### Step 3: Run the Application
```bash
docker compose -f docker-compose.fast.yml up
```

---

## File Overview

| File | Purpose |
|------|---------|
| `Dockerfile.tdlib` | Builds TDLib base image (one-time) |
| `Dockerfile.fast` | Fast rebuild using pre-built TDLib |
| `Dockerfile.cached` | Original (rebuilds everything) |
| `docker-compose.fast.yml` | Uses fast Dockerfile |
| `docker-compose.yml` | Original (uses cached Dockerfile) |
| `build-tdlib-base.sh` | Helper script to build TDLib base |

---

## Build Times Comparison

| Method | First Build | Code Changes |
|--------|-------------|--------------|
| Original `Dockerfile.cached` | ~10 minutes | ~10 minutes ❌ |
| **New Fast Method** | 10 min (one-time) | ~30 seconds ✅ |

---

## Complete Workflow

### First Time Setup
```bash
# 1. Build TDLib base image (one-time, ~10 minutes)
./build-tdlib-base.sh

# 2. Build your application (~30 seconds)
docker compose -f docker-compose.fast.yml build

# 3. Run
docker compose -f docker-compose.fast.yml up
```

### Every Time You Change Code
```bash
# Just rebuild and run (only ~30 seconds!)
docker compose -f docker-compose.fast.yml build
docker compose -f docker-compose.fast.yml up
```

---

## Tips

1. **Keep the TDLib base image**: Don't delete `tdlib-base:1.8.19` image
2. **Use `docker-compose.fast.yml`** for all your work
3. **Original `docker-compose.yml`** is still there if you need self-contained builds

## Cleanup Old Images (Optional)
If you built with the old method and want to save space:
```bash
docker image prune -a
```
But keep `tdlib-base:1.8.19`!

---

## Current Build Status

After running the one-time TDLib base build, you can enjoy:
- ✅ Fast code rebuilds (~30 seconds)
- ✅ No more waiting for TDLib compilation
- ✅ Same functionality, faster development
