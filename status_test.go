package status

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
)

// mockDatabase implements a minimal database.Database interface for testing
type mockDatabase struct {
	pingError error
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	return m.pingError
}

func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	return nil
}

func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	return nil, nil
}

func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
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

func TestStatusPlugin_Name(t *testing.T) {
	plugin := NewPlugin()
	if name := plugin.Name(); name != "status" {
		t.Errorf("expected plugin name 'status', got '%s'", name)
	}
}

func TestStatusPlugin_Initialize(t *testing.T) {
	plugin := NewPlugin().(*StatusPlugin)

	config := map[string]interface{}{
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

func TestStatusPlugin_StatusCheckWithDatabase(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*StatusPlugin)

	// Initialize with healthy database
	config := map[string]interface{}{
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

func TestStatusPlugin_StatusCheckDatabaseDown(t *testing.T) {
	app := fiber.New()
	plugin := &StatusPlugin{
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

func TestStatusPlugin_StatusCheckNoDatabase(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*StatusPlugin)

	// Initialize without database
	config := map[string]interface{}{}
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

func TestStatusPlugin_CustomEndpoint(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*StatusPlugin)

	// Initialize with custom endpoint
	config := map[string]interface{}{
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

func TestStatusPlugin_ServerConfigParsing(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedScheme string
		expectedHost   string
		expectedPort   int
	}{
		{
			name:           "default values when no config",
			config:         map[string]interface{}{},
			expectedScheme: "http",
			expectedHost:   "localhost",
			expectedPort:   8000,
		},
		{
			name: "server config from GoREST injection",
			config: map[string]interface{}{
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
			config: map[string]interface{}{
				"server_port": 3000,
			},
			expectedScheme: "http",
			expectedHost:   "localhost",
			expectedPort:   3000,
		},
		{
			name: "custom scheme only",
			config: map[string]interface{}{
				"server_scheme": "https",
			},
			expectedScheme: "https",
			expectedHost:   "localhost",
			expectedPort:   8000,
		},
		{
			name: "custom host only",
			config: map[string]interface{}{
				"server_host": "0.0.0.0",
			},
			expectedScheme: "http",
			expectedHost:   "0.0.0.0",
			expectedPort:   8000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := NewPlugin().(*StatusPlugin)

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
