# Minimax Telegram Bot

A Telegram bot that integrates with Minimax 2.1 AI for conversational AI and interactive content creation.

## Features

### Conversational AI
- Natural language conversations with Minimax 2.1
- Conversation history maintained per user
- Context-aware responses

### Content Creation Wizards
Interactive multi-step wizards for creating various content types:

| Command | Description |
|---------|-------------|
| `/create marketing` | Marketing copy with 8 guided questions |
| `/create email` | Email content with 6 guided questions |
| `/create report` | Business report with 7 guided questions |
| `/create script` | Video/podcast script with 7 guided questions |
| `/create whitepaper` | Whitepaper with 7 guided questions |
| `/create story` | Creative story with 6 guided questions |
| `/create poem` | Poem with 5 guided questions |

### Quick Mode with Flags
Bypass the wizard and generate content directly:

```
/create marketing -t "promote my new app"
/create email -t "newsletter signup" -s "friendly"
/create story -q
```

Available flags:
- `-t <text>` - Quick prompt
- `-m <text>` - Additional instructions
- `-s <style>` - Writing style
- `-q` - Quick mode

### Bot Commands
- `/start` - Welcome message
- `/help` - Show help information
- `/clear` - Clear conversation history
- `/status` - Show bot status
- `/cancel` - Cancel active wizard

## Prerequisites

- Go 1.21 or later
- Telegram Bot Token (from @BotFather)
- Minimax API Key

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd telegram-bot
```

2. Install dependencies:
```bash
go mod download
```

3. Create configuration file:
```bash
cp .env.example .env
```

4. Edit `.env` with your credentials:
```env
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
MINIMAX_API_KEY=your_minimax_api_key
MINIMAX_MODEL=abab6.5s-chat
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API token | Required |
| `MINIMAX_API_KEY` | Minimax API key | Required |
| `MINIMAX_MODEL` | Minimax model to use | `abab6.5s-chat` |
| `MAX_MESSAGE_LENGTH` | Max message length in characters | `4096` |
| `POLLING_TIMEOUT` | Polling timeout in seconds | `60` |
| `ENABLE_INLINE_MODE` | Enable inline mode | `false` |

## Running

### Development
```bash
go run cmd/telegram-bot/main.go
```

### Production
```bash
go build -o telegram-bot cmd/telegram-bot/main.go
./telegram-bot
```

## Project Structure

```
telegram-bot/
├── cmd/
│   └── telegram-bot/
│       └── main.go              # Application entry point
├── internal/
│   ├── handler/
│   │   └── handler.go          # Message handling logic
│   ├── minimax/
│   │   └── client.go           # Minimax API client
│   ├── telegram/
│   │   └── client.go           # Telegram API client
│   └── wizard/
│       └── wizard.go           # Content creation wizards
├── pkg/
│   ├── config/
│   │   └── config.go           # Configuration management
│   └── logger/
│       └── logger.go            # Logging utilities
├── .env.example                # Environment template
├── go.mod                      # Go module definition
└── README.md                   # This file
```

## Architecture

### Telegram Client
- Long polling for receiving updates
- Message sending with rate limiting
- Support for commands and regular messages

### Minimax Client
- Chat completion API integration
- Per-user conversation history
- Configurable model selection

### Wizard System
- Interactive multi-step sessions
- Timeout handling (10 minutes)
- State management per user

## Testing

Run all tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -cover ./...
```

## License

MIT License
