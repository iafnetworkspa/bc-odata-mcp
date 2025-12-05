package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/iafnetworkspa/bc-odata-mcp/internal/bc"
	"github.com/iafnetworkspa/bc-odata-mcp/internal/mcp"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (optional, uses environment variables by default)")
	flag.Parse()

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create and run MCP server
	server, err := mcp.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating server: %v\n", err)
		os.Exit(1)
	}

	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}

// loadConfig loads configuration from environment variables
func loadConfig(configPath string) (bc.Config, error) {
	cfg := bc.Config{
		GrantType:    getEnv("BC_GRANT_TYPE", "client_credentials"),
		ClientID:     getEnv("BC_CLIENT_ID", ""),
		ClientSecret: getEnv("BC_CLIENT_SECRET", ""),
		ScopeAPI:     getEnv("BC_SCOPE_API", ""),
		TokenURL:     getEnv("BC_TOKEN_URL", ""),
		ContentType:  getEnv("BC_CONTENT_TYPE", "application/x-www-form-urlencoded"),
		BasePath:     getEnv("BC_BASE_PATH", ""),
		TenantID:     getEnv("BC_TENANT_ID", ""),
		Environment:  getEnv("BC_ENVIRONMENT", "Production"),
		Company:      getEnv("BC_COMPANY", ""),
		APITimeout:   getEnvInt("BC_API_TIMEOUT", 90),
	}

	// Validate required fields
	if cfg.ClientID == "" {
		return cfg, fmt.Errorf("BC_CLIENT_ID is required")
	}
	if cfg.ClientSecret == "" {
		return cfg, fmt.Errorf("BC_CLIENT_SECRET is required")
	}
	if cfg.ScopeAPI == "" {
		return cfg, fmt.Errorf("BC_SCOPE_API is required")
	}
	if cfg.TokenURL == "" {
		return cfg, fmt.Errorf("BC_TOKEN_URL is required")
	}
	if cfg.BasePath == "" {
		return cfg, fmt.Errorf("BC_BASE_PATH is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}

