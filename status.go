package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/plugin"
)

// defaultHealthCacheTTL bounds how long a database Ping result is reused between
// probes. It defaults short so a transient outage surfaces almost immediately.
const defaultHealthCacheTTL = time.Second

// Plugin provides a status check endpoint for health monitoring
type Plugin struct {
	db       database.Database
	endpoint string
	scheme   string
	host     string
	port     int
	config   map[string]any

	// healthTTL is the window during which a Ping result is served from cache.
	// A zero value disables caching so every request pings the database.
	healthTTL time.Duration

	healthMu     sync.Mutex
	healthExpiry time.Time
	healthErr    error
}

// NewPlugin creates a new instance with default settings
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin identifier
func (p *Plugin) Name() string {
	return "status"
}

// Initialize configures the plugin with the provided configuration map
func (p *Plugin) Initialize(config map[string]any) error {
	p.config = config

	// Log full config for debugging
	logger.Log.Info("Status plugin Initialize called", "config_keys", getConfigKeys(config))

	if db, ok := config["database"].(database.Database); ok {
		p.db = db
	}

	p.healthTTL = defaultHealthCacheTTL
	if ttl, ok := config["health_cache_ttl"].(time.Duration); ok && ttl >= 0 {
		p.healthTTL = ttl
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

func getConfigKeys(config map[string]any) []string {
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	return keys
}

// Handler returns a no-op middleware handler (status endpoint is registered via SetupEndpoints)
func (p *Plugin) Handler() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Next()
	}
}

// SetupEndpoints registers the status check HTTP endpoint
func (p *Plugin) SetupEndpoints(router fiber.Router) error {
	logger.Log.Debug("Registering status endpoint", "path", "/"+p.endpoint)
	router.Get("/"+p.endpoint, p.statusCheckHandler())

	port := fmt.Sprintf("%d", p.port)

	url := p.scheme + "://" + p.host
	if (p.scheme == "http" && p.port != 80) ||
		(p.scheme == "https" && p.port != 443) {
		url += ":" + port
	}

	logger.Log.Info("Health check available", "url", fmt.Sprintf("%s/%s", url, p.endpoint))

	return nil
}

func (p *Plugin) statusCheckHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
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

		if err := p.cachedPing(ctx); err != nil {
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

// cachedPing returns the most recent database Ping result while it is still
// within healthTTL. The lock is intentionally held across the Ping so a burst
// of probes racing an expired entry collapses into a single round-trip: the
// first goroutine refreshes, the rest observe the fresh entry and return.
func (p *Plugin) cachedPing(ctx context.Context) error {
	p.healthMu.Lock()
	defer p.healthMu.Unlock()

	if p.healthTTL > 0 && time.Now().Before(p.healthExpiry) {
		return p.healthErr
	}

	err := p.db.Ping(ctx)
	if p.healthTTL > 0 {
		p.healthErr = err
		p.healthExpiry = time.Now().Add(p.healthTTL)
	}
	return err
}
