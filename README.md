# 🌸 Botagul

**AI-powered Telegram bot for florists in Kazakhstan.**

Botagul turns chaotic Telegram messages into structured orders — no forms, no CRM fields, just natural conversation. Customers write in a mix of Russian and Kazakh slang, and the bot extracts order details using Google Gemini.

## How It Works

```
Customer → Telegram → Botagul → Gemini LLM → PostgreSQL
```

1. Customer sends a message: *"Салем! Мне нужно 15 пионов на 8 марта, бюджет 20к, доставка в Алмалинский район"*
2. Telegram delivers it to the bot's webhook
3. Gemini extracts structured data (items, budget, delivery date, notes)
4. The order is saved to PostgreSQL as JSONB
5. Customer gets a beautiful confirmation ✅

## Architecture

```
botagul/
├── cmd/bot/main.go              # Entry point, wiring, graceful shutdown
├── internal/
│   ├── config/config.go         # Env-based configuration
│   ├── domain/
│   │   ├── context.go           # Intent, OrderContext, ContextExtractor
│   │   └── order.go             # Order entity, Repository interface
│   ├── llm/
│   │   ├── client.go            # Gemini API client (contents/parts format)
│   │   └── extractor.go         # LLM-based order extraction with date injection
│   ├── repository/
│   │   └── postgres.go          # PostgreSQL + JSONB storage
│   └── telegram/
│       ├── api.go               # Telegram Bot API client
│       └── handler.go           # Webhook handler with chat ID filtering
├── migrations/
│   └── 001_create_orders.sql    # Database schema
├── .env.example                 # Environment template
└── go.mod
```

## Prerequisites

- **Go** 1.21+
- **PostgreSQL** 17+
- **Telegram Bot Token** — get one from [@BotFather](https://t.me/BotFather)
- **Gemini API Key** — get one from [Google AI Studio](https://aistudio.google.com/apikey)
- **ngrok** (for local development) — [ngrok.com](https://ngrok.com)

## Quick Start

### 1. Clone & configure

```bash
git clone https://github.com/Jakisheff/botagul.git
cd botagul
cp .env.example .env
```

Edit `.env` with your values:

```env
TELEGRAM_TOKEN=123456:ABC-DEF...
GEMINI_API_KEY=AIza...
DATABASE_URL=postgres://youruser@localhost:5432/botagul?sslmode=disable
ALLOWED_CHAT_IDS=123456789    # Your Telegram chat ID
```

### 2. Set up the database

```bash
createdb botagul
```

The app auto-migrates on startup — no manual SQL needed.

### 3. Run the bot

```bash
go run ./cmd/bot/
```

You should see:

```
[init] ✅ connected to PostgreSQL
[init] ✅ migration applied
[init] ✅ LLM configured (model: gemini-2.0-flash)
[server] 🚀 botagul listening on :8080
```

### 4. Expose to Telegram (local dev)

```bash
ngrok http 8080
```

Register the webhook:

```bash
curl "https://api.telegram.org/bot<YOUR_TOKEN>/setWebhook?url=<NGROK_URL>/webhook"
```

### 5. Test it!

Send your bot a message like:

> Здравствуйте! Хочу заказать букет из 25 красных роз на день рождения жены. Бюджет до 30 тысяч, нужна доставка 1 марта к 18:00 в район Байконура. Оплата каспи.

Check the database:

```bash
psql botagul -c "SELECT id, chat_id, context->>'intent' AS intent, context->>'items' AS items, context->>'budget' AS budget FROM orders;"
```

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TELEGRAM_TOKEN` | ✅ | — | Bot token from @BotFather |
| `GEMINI_API_KEY` | ✅ | — | Google AI API key |
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `GEMINI_MODEL` | ❌ | `gemini-2.0-flash` | Gemini model to use |
| `ALLOWED_CHAT_IDS` | ❌ | *all* | Comma-separated whitelist of chat IDs |
| `PORT` | ❌ | `8080` | HTTP server port |

## Extracted Order Fields

The LLM extracts the following from each message:

| Field | Type | Example |
|-------|------|---------|
| `intent` | `new_order` / `question` / `other` | `"new_order"` |
| `client_name` | string | `"Айгуль"` |
| `items` | string | `"25 красных роз"` |
| `budget` | int (₸) | `30000` |
| `delivery_date` | ISO 8601 | `"2026-03-01T18:00:00+05:00"` |
| `notes` | string | `"оплата каспи, без упаковки"` |

## Key Design Decisions

- **Gemini `system_instruction`** — system prompt uses the native field (not a user message)
- **Dynamic date injection** — current Almaty time is injected into every prompt so the LLM can resolve relative dates ("завтра", "в пятницу")
- **Instant 200 OK** — webhook responds immediately; processing runs in a goroutine to avoid Telegram retries
- **JSONB storage** — `OrderContext` is stored as JSONB for flexible querying without rigid schema migrations

## Health Check

```bash
curl http://localhost:8080/health
# ok
```

## License

MIT
