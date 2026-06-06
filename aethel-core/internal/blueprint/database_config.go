package blueprint

type DatabaseConfig struct {
	Metadata               Metadata                     `yaml:"metadata"`
	GlobalDatabaseDefaults GlobalDatabaseDefaults       `yaml:"global_database_defaults"`
	Environments           map[string]EnvironmentConfig `yaml:"environments"`
	Schema                 SchemaConfig                 `yaml:"schema"`
	Partitioning           map[string]PartitionConfig   `yaml:"partitioning"`
	Extensions             ExtensionConfig              `yaml:"extensions"`
	Performance            PerformanceConfig            `yaml:"performance"`
	Server                 ServerConfig                 `yaml:"server"`
	Auth                   AuthConfig                   `yaml:"auth"`
}

type Metadata struct {
	Version          string `yaml:"version"`
	EngineTarget     string `yaml:"engine_target"`
	StrictValidation bool   `yaml:"strict_validation"`
}

type GlobalDatabaseDefaults struct {
	Dialect  string          `yaml:"dialect"`
	Encoding string          `yaml:"encoding"`
	Timezone string          `yaml:"timezone"`
	Logging  LoggingDefaults `yaml:"logging"`
}

type LoggingDefaults struct {
	LogQueries            bool   `yaml:"log_queries"`
	SlowQueryThresholdMs  int    `yaml:"slow_query_threshold_ms"`
	LogLevel              string `yaml:"log_level"`
}

type EnvironmentConfig struct {
	Connection ConnectionConfig `yaml:"connection"`
	Pooling    PoolingConfig    `yaml:"pooling"`
	Migrations MigrationConfig  `yaml:"migrations"`
}

type ConnectionConfig struct {
	Host                string `yaml:"host"`
	Port                int    `yaml:"port"`
	Database            string `yaml:"database"`
	User                string `yaml:"user"`
	SSLMode             string `yaml:"ssl_mode"`
	SSLRootCertPath     string `yaml:"ssl_root_cert_path"`
	ConnectionStringEnv string `yaml:"connection_string_env"`
}

type PoolingConfig struct {
	MaxOpenConnections           int `yaml:"max_open_connections"`
	MaxIdleConnections           int `yaml:"max_idle_connections"`
	ConnectionMaxLifetimeMinutes int `yaml:"connection_max_lifetime_minutes"`
	ConnectionMaxIdleTimeMinutes int `yaml:"connection_max_idle_time_minutes"`
}

type MigrationConfig struct {
	Directory          string `yaml:"directory"`
	AutoRunOnStartup   bool   `yaml:"auto_run_on_startup"`
	TableName          string `yaml:"table_name"`
	LockTimeoutSeconds int    `yaml:"lock_timeout_seconds"`
}

type SchemaConfig struct {
	DefaultSchema string            `yaml:"default_schema"`
	NameAliases   map[string]string `yaml:"name_aliases"`
	EnumAliases   map[string]string `yaml:"enum_aliases"`
}

type PartitionConfig struct {
	Type            string          `yaml:"type"`
	Column          string          `yaml:"column"`
	Interval        string          `yaml:"interval"`
	RetentionPolicy RetentionPolicy `yaml:"retention_policy"`
}

type RetentionPolicy struct {
	Enabled       bool `yaml:"enabled"`
	RetainMonths  int  `yaml:"retain_months"`
}

type ExtensionConfig struct {
	Required []string `yaml:"required"`
	Optional []string `yaml:"optional"`
}

type PerformanceConfig struct {
	StatementTimeoutMs           int `yaml:"statement_timeout_ms"`
	IdleInTransactionTimeoutMs   int `yaml:"idle_in_transaction_timeout_ms"`
	LockTimeoutMs                int `yaml:"lock_timeout_ms"`
}

// ServerConfig holds HTTP-level guardrails loaded from the server: blueprint section.
// These are applied at startup by the Go backend middleware stack.
type ServerConfig struct {
	// RateLimitRPM is the global per-IP token bucket refill rate (requests per minute).
	// Default (if not set in blueprint): 600.
	RateLimitRPM int `yaml:"rate_limit_rpm"`
	// RateLimitBurst is the maximum burst capacity above the steady refill rate.
	// Default (if not set in blueprint): 100.
	RateLimitBurst int `yaml:"rate_limit_burst"`
	// BodyLimitBytes is the maximum accepted request body size in bytes.
	// Default (if not set in blueprint): 1 MiB (1048576).
	BodyLimitBytes int64 `yaml:"body_limit_bytes"`
}

// ServerDefaults returns safe production defaults for ServerConfig fields
// that were not explicitly set in the blueprint (zero-value check).
func (s ServerConfig) ServerDefaults() ServerConfig {
	if s.RateLimitRPM <= 0 {
		s.RateLimitRPM = 600
	}
	if s.RateLimitBurst <= 0 {
		s.RateLimitBurst = 100
	}
	if s.BodyLimitBytes <= 0 {
		s.BodyLimitBytes = 1048576 // 1 MiB
	}
	return s
}

// AuthConfig holds Argon2id cost parameters and token TTLs that IT administrators
// can tune in server-database.yaml. All fields default to safe production values
// when omitted or set to zero.
type AuthConfig struct {
	Argon2MemoryKiB     uint32 `yaml:"argon2_memory_kib"`
	Argon2Iterations    uint32 `yaml:"argon2_iterations"`
	Argon2Parallelism   uint8  `yaml:"argon2_parallelism"`
	AccessTokenTTLMin   int    `yaml:"access_token_ttl_min"`
	RefreshTokenTTLDays int    `yaml:"refresh_token_ttl_days"`
}

// SetDefaults fills zero-value fields with safe production defaults.
func (a *AuthConfig) SetDefaults() {
	if a.Argon2MemoryKiB == 0 {
		a.Argon2MemoryKiB = 65536
	}
	if a.Argon2Iterations == 0 {
		a.Argon2Iterations = 3
	}
	if a.Argon2Parallelism == 0 {
		a.Argon2Parallelism = 4
	}
	if a.AccessTokenTTLMin == 0 {
		a.AccessTokenTTLMin = 30
	}
	if a.RefreshTokenTTLDays == 0 {
		a.RefreshTokenTTLDays = 30
	}
}
