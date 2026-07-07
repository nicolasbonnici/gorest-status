package status

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/database"
)

// mockDatabase implements a minimal database.Database interface for testing
type mockDatabase struct {
	pingError error
	pingCount atomic.Int32
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	m.pingCount.Add(1)
	return m.pingError
}

func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return nil
}

func (m *mockDatabase) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	return nil, nil
}

func (m *mockDatabase) Exec(ctx context.Context, query string, args ...any) (database.Result, error) {
	return nil, nil
}

func (m *mockDatabase) Close() error {
	return nil
}

func (m *mockDatabase) Dialect() database.Dialect {
	return nil
}

func (m *mockDatabase) Begin(ctx context.Context) (database.Tx, error) {
	return nil, nil
}

func (m *mockDatabase) Connect(ctx context.Context, connStr string) error {
	return nil
}

func (m *mockDatabase) DriverName() string {
	return "mock"
}

type mockIntrospector struct{}

func (m *mockIntrospector) LoadSchema(ctx context.Context) ([]database.TableSchema, error) {
	return nil, nil
}

func (m *mockIntrospector) GetColumns(ctx context.Context, tableName string) ([]database.Column, error) {
	return nil, nil
}

func (m *mockIntrospector) GetRelations(ctx context.Context) ([]database.Relation, error) {
	return nil, nil
}

func (m *mockDatabase) Introspector() database.SchemaIntrospector {
	return &mockIntrospector{}
}

func TestPlugin_Name(t *testing.T) {
	plugin := NewPlugin()
	if name := plugin.Name(); name != "status" {
		t.Errorf("expected plugin name 'status', got '%s'", name)
	}
}

func TestPlugin_Initialize(t *testing.T) {
	plugin := NewPlugin().(*Plugin)

	config := map[string]any{
		"database": &mockDatabase{},
	}

	err := plugin.Initialize(config)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	if plugin.db == nil {
		t.Error("expected database to be set")
	}
}

func TestPlugin_StatusCheckWithDatabase(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*Plugin)

	// Initialize with healthy database
	config := map[string]any{
		"database": &mockDatabase{pingError: nil},
	}
	if err := plugin.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := plugin.SetupEndpoints(app); err != nil {
		t.Fatalf("SetupEndpoints failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPlugin_StatusCheckDatabaseDown(t *testing.T) {
	app := fiber.New()
	plugin := &Plugin{
		db:       &mockDatabase{pingError: errors.New("connection failed")},
		endpoint: "status",
		scheme:   "http",
		host:     "localhost",
		port:     8000,
	}
	if err := plugin.SetupEndpoints(app); err != nil {
		t.Fatalf("SetupEndpoints failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}
}

func TestPlugin_StatusCheckNoDatabase(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*Plugin)

	// Initialize without database
	config := map[string]any{}
	if err := plugin.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := plugin.SetupEndpoints(app); err != nil {
		t.Fatalf("SetupEndpoints failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPlugin_CustomEndpoint(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*Plugin)

	// Initialize with custom endpoint
	config := map[string]any{
		"endpoint": "health",
	}
	if err := plugin.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := plugin.SetupEndpoints(app); err != nil {
		t.Fatalf("SetupEndpoints failed: %v", err)
	}

	// Test that custom endpoint works
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Test that default endpoint doesn't work
	req = httptest.NewRequest("GET", "/status", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("expected status 404 for default endpoint, got %d", resp.StatusCode)
	}
}

func TestPlugin_HealthCacheCoalescesPings(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{}
	plugin := NewPlugin().(*Plugin)
	if err := plugin.Initialize(map[string]any{"database": db}); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := plugin.SetupEndpoints(app); err != nil {
		t.Fatalf("SetupEndpoints failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/status", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
	}

	if got := db.pingCount.Load(); got != 1 {
		t.Errorf("expected 1 ping for a burst within TTL, got %d", got)
	}
}

func TestPlugin_HealthCacheExpires(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{}
	plugin := NewPlugin().(*Plugin)
	if err := plugin.Initialize(map[string]any{
		"database":         db,
		"health_cache_ttl": time.Duration(0),
	}); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := plugin.SetupEndpoints(app); err != nil {
		t.Fatalf("SetupEndpoints failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/status", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
	}

	if got := db.pingCount.Load(); got != 3 {
		t.Errorf("expected 3 pings with caching disabled, got %d", got)
	}
}

func TestPlugin_ServerConfigParsing(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]any
		expectedScheme string
		expectedHost   string
		expectedPort   int
	}{
		{
			name:           "default values when no config",
			config:         map[string]any{},
			expectedScheme: "http",
			expectedHost:   "localhost",
			expectedPort:   8000,
		},
		{
			name: "server config from GoREST injection",
			config: map[string]any{
				"server_scheme": "https",
				"server_host":   "api.example.com",
				"server_port":   443,
			},
			expectedScheme: "https",
			expectedHost:   "api.example.com",
			expectedPort:   443,
		},
		{
			name: "partial config with defaults",
			config: map[string]any{
				"server_port": 3000,
			},
			expectedScheme: "http",
			expectedHost:   "localhost",
			expectedPort:   3000,
		},
		{
			name: "custom scheme only",
			config: map[string]any{
				"server_scheme": "https",
			},
			expectedScheme: "https",
			expectedHost:   "localhost",
			expectedPort:   8000,
		},
		{
			name: "custom host only",
			config: map[string]any{
				"server_host": "0.0.0.0",
			},
			expectedScheme: "http",
			expectedHost:   "0.0.0.0",
			expectedPort:   8000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := NewPlugin().(*Plugin)

			if err := plugin.Initialize(tt.config); err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}

			if plugin.scheme != tt.expectedScheme {
				t.Errorf("expected scheme %s, got %s", tt.expectedScheme, plugin.scheme)
			}
			if plugin.host != tt.expectedHost {
				t.Errorf("expected host %s, got %s", tt.expectedHost, plugin.host)
			}
			if plugin.port != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, plugin.port)
			}
		})
	}
}
