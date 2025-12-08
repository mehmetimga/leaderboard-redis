package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Redis       RedisConfig       `yaml:"redis"`
	Postgres    PostgresConfig    `yaml:"postgres"`
	Kafka       KafkaConfig       `yaml:"kafka"`
	Sync        SyncConfig        `yaml:"sync"`
	Leaderboard LeaderboardConfig `yaml:"leaderboard"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr         string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxConnections  int           `yaml:"max_connections"`
	MinConnections  int           `yaml:"min_connections"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
}

// ConnectionString returns the PostgreSQL connection string
func (c *PostgresConfig) ConnectionString() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, sslMode,
	)
}

// KafkaConfig holds Kafka connection configuration
type KafkaConfig struct {
	Brokers       []string      `yaml:"brokers"`
	Topic         string        `yaml:"topic"`
	GroupID       string        `yaml:"group_id"`
	Enabled       bool          `yaml:"enabled"`
	BatchSize     int           `yaml:"batch_size"`
	BatchTimeout  time.Duration `yaml:"batch_timeout"`
	RetryAttempts int           `yaml:"retry_attempts"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
}

// SyncConfig holds synchronization worker configuration
type SyncConfig struct {
	Interval  time.Duration `yaml:"interval"`
	BatchSize int           `yaml:"batch_size"`
	Enabled   bool          `yaml:"enabled"`
}

// LeaderboardConfig holds leaderboard-specific configuration
type LeaderboardConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	MaxLimit     int `yaml:"max_limit"`
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	return &cfg, nil
}

// applyDefaults sets default values for missing configuration
func (c *Config) applyDefaults() {
	// Server defaults
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 5 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 10 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 120 * time.Second
	}

	// Redis defaults
	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}
	if c.Redis.PoolSize == 0 {
		c.Redis.PoolSize = 100
	}
	if c.Redis.MinIdleConns == 0 {
		c.Redis.MinIdleConns = 10
	}
	if c.Redis.DialTimeout == 0 {
		c.Redis.DialTimeout = 5 * time.Second
	}
	if c.Redis.ReadTimeout == 0 {
		c.Redis.ReadTimeout = 3 * time.Second
	}
	if c.Redis.WriteTimeout == 0 {
		c.Redis.WriteTimeout = 3 * time.Second
	}

	// PostgreSQL defaults
	if c.Postgres.Host == "" {
		c.Postgres.Host = "localhost"
	}
	if c.Postgres.Port == 0 {
		c.Postgres.Port = 5432
	}
	if c.Postgres.MaxConnections == 0 {
		c.Postgres.MaxConnections = 50
	}
	if c.Postgres.MinConnections == 0 {
		c.Postgres.MinConnections = 5
	}
	if c.Postgres.MaxConnLifetime == 0 {
		c.Postgres.MaxConnLifetime = 1 * time.Hour
	}
	if c.Postgres.MaxConnIdleTime == 0 {
		c.Postgres.MaxConnIdleTime = 30 * time.Minute
	}

	// Kafka defaults
	if len(c.Kafka.Brokers) == 0 {
		c.Kafka.Brokers = []string{"localhost:9092"}
	}
	if c.Kafka.Topic == "" {
		c.Kafka.Topic = "leaderboard-scores"
	}
	if c.Kafka.GroupID == "" {
		c.Kafka.GroupID = "leaderboard-consumer"
	}
	if c.Kafka.BatchSize == 0 {
		c.Kafka.BatchSize = 100
	}
	if c.Kafka.BatchTimeout == 0 {
		c.Kafka.BatchTimeout = 1 * time.Second
	}
	if c.Kafka.RetryAttempts == 0 {
		c.Kafka.RetryAttempts = 3
	}
	if c.Kafka.RetryDelay == 0 {
		c.Kafka.RetryDelay = 1 * time.Second
	}

	// Sync defaults
	if c.Sync.Interval == 0 {
		c.Sync.Interval = 30 * time.Minute
	}
	if c.Sync.BatchSize == 0 {
		c.Sync.BatchSize = 1000
	}

	// Leaderboard defaults
	if c.Leaderboard.DefaultLimit == 0 {
		c.Leaderboard.DefaultLimit = 100
	}
	if c.Leaderboard.MaxLimit == 0 {
		c.Leaderboard.MaxLimit = 1000
	}
}

// DefaultConfig returns a configuration with all defaults
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	cfg.Sync.Enabled = true
	return cfg
}

