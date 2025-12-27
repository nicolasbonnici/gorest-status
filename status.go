package status

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/plugin"
)

// StatusPlugin provides a status check endpoint
type StatusPlugin struct {
	db       database.Database
	endpoint string
	config   map[string]interface{}
}

func NewPlugin() plugin.Plugin {
	return &StatusPlugin{}
}

func (p *StatusPlugin) Name() string {
	return "status"
}

func (p *StatusPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	if db, ok := config["database"].(database.Database); ok {
		p.db = db
	}
	if endpoint, ok := config["endpoint"].(string); ok {
		p.endpoint = endpoint
		logger.Log.Debug("Status plugin using custom endpoint from config", "endpoint", endpoint)
	} else {
		p.endpoint = "status" // default endpoint
		logger.Log.Debug("Status plugin using default endpoint", "endpoint", p.endpoint)
	}
	return nil
}

// Handler returns a no-op middleware since status endpoint is set up via SetupEndpoints
func (p *StatusPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// SetupEndpoints implements the optional EndpointSetup interface
func (p *StatusPlugin) SetupEndpoints(app *fiber.App) error {
	// Detect port from app configuration
	port := p.detectPort(app)

	logger.Log.Debug("Registering status endpoint", "path", "/"+p.endpoint, "port", port)
	app.Get("/"+p.endpoint, p.statusCheckHandler())
	logger.Log.Info("Health check available", "url", fmt.Sprintf("http://localhost:%s/%s", port, p.endpoint))
	return nil
}

// detectPort attempts to detect the port from various sources
func (p *StatusPlugin) detectPort(app *fiber.App) string {
	// 1. Check if port was provided in plugin config (backwards compatibility)
	if p.config != nil {
		if port, ok := p.config["port"].(string); ok && port != "" {
			return port
		}
	}

	// 2. Try to detect from environment variable PORT
	if port := os.Getenv("PORT"); port != "" {
		return port
	}

	// 3. Try to detect from GOREST_PORT environment variable
	if port := os.Getenv("GOREST_PORT"); port != "" {
		return port
	}

	// 4. Check Fiber app config for Network address
	fiberConfig := app.Config()
	if fiberConfig.Network != "" {
		// Network format could be ":port" or "host:port"
		parts := strings.Split(fiberConfig.Network, ":")
		if len(parts) > 1 && parts[len(parts)-1] != "" {
			return parts[len(parts)-1]
		}
	}

	// 5. Default to 8080
	return "8080"
}

// statusCheckHandler creates the status check handler
func (p *StatusPlugin) statusCheckHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Perform status check
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		if p.db == nil {
			return c.JSON(fiber.Map{
				"status": "healthy",
				"database": fiber.Map{
					"status": "not_configured",
				},
			})
		}

		if err := p.db.Ping(ctx); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "unhealthy",
				"database": fiber.Map{
					"status": "down",
					"error":  err.Error(),
				},
			})
		}

		return c.JSON(fiber.Map{
			"status": "healthy",
			"database": fiber.Map{
				"status": "up",
			},
		})
	}
}
