# GoREST Status Plugin

[![CI](https://github.com/nicolasbonnici/gorest-status/actions/workflows/ci.yml/badge.svg)](https://github.com/nicolasbonnici/gorest-status/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nicolasbonnici/gorest-status)](https://goreportcard.com/report/github.com/nicolasbonnici/gorest-status)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Status check endpoint plugin for GoREST framework.

## Installation

```bash
go get github.com/nicolasbonnici/gorest-status
```

## Usage

```go
import (
	"github.com/nicolasbonnici/gorest/pluginloader"
	statusplugin "github.com/nicolasbonnici/gorest-status"
)

func init() {
	pluginloader.RegisterPluginFactory("status", statusplugin.NewPlugin)
}
```

### Configuration

Add to your `gorest.yaml`:

```yaml
plugins:
  - name: status
    enabled: true
```

#### Advanced Configuration

You can customize the endpoint path using plugin configuration:

```yaml
plugins:
  - name: status
    enabled: true
    config:
      endpoint: "health"  # Custom endpoint path (default: "status")
```

**Configuration Parameters:**

- `endpoint` (string): Custom path for the health check endpoint. Default: `"status"`
  - Example: `"health"` creates endpoint at `/health`
  - Example: `"api/status"` creates endpoint at `/api/status`

**Port Detection:**

The plugin automatically detects the server port for logging purposes using the following priority:
1. `port` parameter in plugin config (if provided)
2. `PORT` environment variable
3. `GOREST_PORT` environment variable
4. Fiber app configuration
5. Default: `8080`

Example with explicit port:
```yaml
plugins:
  - name: status
    enabled: true
    config:
      endpoint: "health"
      port: "3000"  # Optional: override automatic detection
```

## Features

- Customizable status check endpoint (default: `/status`)
- Database connectivity check
- Returns HTTP 200 when healthy
- Returns HTTP 503 when database is down
- JSON response format
- Startup logging with endpoint URL

## Response Format

Healthy (database configured and up):
```json
{
  "status": "healthy",
  "database": {
    "status": "up"
  }
}
```

Unhealthy (database down):
```json
{
  "status": "unhealthy",
  "database": {
    "status": "down",
    "error": "connection failed"
  }
}
```

Database not configured:
```json
{
  "status": "healthy",
  "database": {
    "status": "not_configured"
  }
}
```

## License

MIT License - see LICENSE file for details
