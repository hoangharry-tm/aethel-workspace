package blueprint

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadDatabaseConfig(path string) (*DatabaseConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read blueprint %s: %w", path, err)
	}
	var cfg DatabaseConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse blueprint %s: %w", path, err)
	}
	cfg.Auth.SetDefaults()
	if err := validateDatabaseConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid blueprint %s: %w", path, err)
	}
	return &cfg, nil
}

func LoadQueriesConfig(path string) (*QueriesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read blueprint %s: %w", path, err)
	}
	var cfg QueriesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse blueprint %s: %w", path, err)
	}
	return &cfg, nil
}

func validateDatabaseConfig(cfg *DatabaseConfig) error {
	if cfg.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}
	if len(cfg.Environments) == 0 {
		return fmt.Errorf("at least one environment is required")
	}
	if cfg.Schema.DefaultSchema == "" {
		return fmt.Errorf("schema.default_schema is required")
	}
	for name, env := range cfg.Environments {
		if env.Connection.Host == "" && env.Connection.ConnectionStringEnv == "" {
			return fmt.Errorf("environment %q: connection.host or connection_string_env is required", name)
		}
	}
	return nil
}
