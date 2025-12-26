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

## Features

- Status check endpoint at `/status`
- Database connectivity check
- Returns HTTP 200 when healthy
- Returns HTTP 503 when database is down
- JSON response format

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
