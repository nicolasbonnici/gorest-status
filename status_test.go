package status

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
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

func TestStatusPlugin_PortDetection(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		envVars      map[string]string
		expectedPort string
	}{
		{
			name:         "default port",
			config:       map[string]interface{}{},
			envVars:      map[string]string{},
			expectedPort: "8080",
		},
		{
			name: "port from config",
			config: map[string]interface{}{
				"port": "3000",
			},
			envVars:      map[string]string{},
			expectedPort: "3000",
		},
		{
			name:   "port from PORT env var",
			config: map[string]interface{}{},
			envVars: map[string]string{
				"PORT": "5000",
			},
			expectedPort: "5000",
		},
		{
			name:   "port from GOREST_PORT env var",
			config: map[string]interface{}{},
			envVars: map[string]string{
				"GOREST_PORT": "9000",
			},
			expectedPort: "9000",
		},
		{
			name: "config takes precedence over env",
			config: map[string]interface{}{
				"port": "4000",
			},
			envVars: map[string]string{
				"PORT": "5000",
			},
			expectedPort: "4000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			app := fiber.New()
			plugin := NewPlugin().(*StatusPlugin)

			if err := plugin.Initialize(tt.config); err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}

			port := plugin.detectPort(app)
			if port != tt.expectedPort {
				t.Errorf("expected port %s, got %s", tt.expectedPort, port)
			}
		})
	}
}
