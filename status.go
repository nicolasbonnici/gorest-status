package status

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

// StatusPlugin provides a status check endpoint
type StatusPlugin struct {
	db database.Database
}

func NewPlugin() plugin.Plugin {
	return &StatusPlugin{}
}

func (p *StatusPlugin) Name() string {
	return "status"
}

func (p *StatusPlugin) Initialize(config map[string]interface{}) error {
	if db, ok := config["database"].(database.Database); ok {
		p.db = db
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
	app.Get("/status", p.statusCheckHandler())
	return nil
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
