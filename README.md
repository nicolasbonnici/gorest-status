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

**Server Configuration:**

The plugin automatically receives server configuration from GoREST v0.4.4+ for accurate URL logging:
- `server_scheme`: Protocol (http/https) - Default: `"http"`
- `server_host`: Hostname or IP address - Default: `"localhost"`
- `server_port`: Port number - Default: `8000`

These parameters are automatically injected by GoREST based on your `gorest.yaml` server configuration. The plugin uses these values to display the correct health check URL in startup logs.

Example server configuration in `gorest.yaml`:
```yaml
server:
  scheme: https
  host: api.example.com
  port: 443

plugins:
  - name: status
    enabled: true
    config:
      endpoint: "health"
```

This will display: `Health check available at https://api.example.com:443/health`

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
