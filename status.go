package status

import (
	"context"
	"fmt"
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
	scheme   string
	host     string
	port     int
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

	// Log full config for debugging
	logger.Log.Info("Status plugin Initialize called", "config_keys", getConfigKeys(config))

	if db, ok := config["database"].(database.Database); ok {
		p.db = db
	}
	if endpoint, ok := config["endpoint"].(string); ok {
		p.endpoint = endpoint
		logger.Log.Info("Status plugin using custom endpoint from config", "endpoint", endpoint)
	} else {
		p.endpoint = "status" // default endpoint
		logger.Log.Info("Status plugin using default endpoint", "endpoint", p.endpoint)
	}

	if scheme, ok := config["server_scheme"].(string); ok && scheme != "" {
		p.scheme = scheme
	} else {
		p.scheme = "http" // default scheme
	}

	if host, ok := config["server_host"].(string); ok && host != "" {
		p.host = host
	} else {
		p.host = "localhost" // default host
	}

	if port, ok := config["server_port"].(int); ok && port > 0 {
		p.port = port
	} else {
		p.port = 8000 // default port
	}

	return nil
}

func getConfigKeys(config map[string]interface{}) []string {
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	return keys
}

func (p *StatusPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

func (p *StatusPlugin) SetupEndpoints(app *fiber.App) error {
	logger.Log.Debug("Registering status endpoint", "path", "/"+p.endpoint)
	app.Get("/"+p.endpoint, p.statusCheckHandler())

	port := fmt.Sprintf("%d", p.port)

	url := p.scheme + "://" + p.host
	if (p.scheme == "http" && p.port != 80) ||
		(p.scheme == "https" && p.port != 443) {
		url += ":" + port
	}

	logger.Log.Info("Health check available", "url", fmt.Sprintf("%s/%s", url, p.endpoint))

	return nil
}

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
