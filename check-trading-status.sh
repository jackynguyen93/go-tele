#!/bin/bash

echo "=== Trading Configuration Check ==="
echo ""

echo "1. Check Settings from Database:"
docker exec telegram-trading-bot sqlite3 /app/data/tdlib.db "SELECT * FROM settings WHERE key LIKE 'trading.%';"

echo ""
echo "2. Check Binance Accounts:"
docker exec telegram-trading-bot sqlite3 /app/data/tdlib.db "SELECT id, name, is_testnet, is_active, is_default FROM binance_accounts;"

echo ""
echo "3. Check Recent Signals:"
docker exec telegram-trading-bot sqlite3 /app/data/tdlib.db "SELECT id, symbol, status, parsed_at FROM signals ORDER BY parsed_at DESC LIMIT 5;"

echo ""
echo "=== How to Fix ==="
echo "If trading.enabled is 'false', enable it via:"
echo "  curl -X PUT http://localhost:3000/api/config -H 'Content-Type: application/json' -d '{\"trading\":{\"enabled\":true}}'"
echo ""
echo "If no Binance accounts exist, add one via the web UI at:"
echo "  http://localhost:3000/accounts"
