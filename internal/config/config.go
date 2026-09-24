package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config wraps all application configuration.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	RabbitMQ RabbitConfig   `mapstructure:"rabbitmq"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Seed     SeedConfig     `mapstructure:"seed"`
	Security SecurityConfig `mapstructure:"security"`
}

// AppConfig contains service-level configuration.
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
	Env         string `mapstructure:"env"`
	Version     string `mapstructure:"version"`
}

// ServerConfig holds HTTP server options.
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout"`
}

// DatabaseConfig describes database connectivity.
type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
}

// RedisConfig configures Redis caching.
type RedisConfig struct {
	Addr     string        `mapstructure:"addr"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	TTL      time.Duration `mapstructure:"ttl"`
}

// RabbitConfig captures RabbitMQ settings.
type RabbitConfig struct {
	URI          string `mapstructure:"uri"`
	ExchangeName string `mapstructure:"exchange"`
	ExchangeType string `mapstructure:"exchange_type"`
	QueueName    string `mapstructure:"queue"`
	RoutingKey   string `mapstructure:"routing_key"`
	Durable      bool   `mapstructure:"durable"`
}

// JWTConfig holds token settings.
type JWTConfig struct {
	PrivateKeyPath string        `mapstructure:"private_key_path"`
	PublicKeyPath  string        `mapstructure:"public_key_path"`
	Issuer         string        `mapstructure:"issuer"`
	Audience       string        `mapstructure:"audience"`
	ExpiresIn      time.Duration `mapstructure:"expires_in"`
	RefreshTTL     time.Duration `mapstructure:"refresh_ttl"`
}

// LoggingConfig configures application logging.
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	Mode  string `mapstructure:"mode"`
}

// SeedConfig controls data seeding.
type SeedConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	AdminEmail    string `mapstructure:"admin_email"`
	AdminPassword string `mapstructure:"admin_password"`
}

// SecurityConfig configures generic security features.
type SecurityConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

// Load reads the config file at the given path applying env overrides.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("GOBASE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("app.name", "go-base")
	v.SetDefault("app.env", "development")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.graceful_timeout", "30s")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.mode", "production")
	v.SetDefault("seed.enabled", false)
	v.SetDefault("seed.admin_email", "")
	v.SetDefault("seed.admin_password", "")

	v.AutomaticEnv()
	for _, key := range []string{
		"app.name", "app.description", "app.env", "app.version",
		"server.host", "server.port", "server.read_timeout", "server.write_timeout",
		"server.idle_timeout", "server.graceful_timeout",
		"database.dsn", "database.max_idle_conns", "database.max_open_conns",
		"database.conn_max_lifetime", "database.auto_migrate",
		"redis.addr", "redis.password", "redis.db", "redis.ttl",
		"rabbitmq.uri", "rabbitmq.exchange", "rabbitmq.exchange_type", "rabbitmq.queue",
		"rabbitmq.routing_key", "rabbitmq.durable",
		"jwt.private_key_path", "jwt.public_key_path", "jwt.issuer", "jwt.audience",
		"jwt.expires_in", "jwt.refresh_ttl",
		"logging.level", "logging.mode", "seed.enabled", "seed.admin_email",
		"seed.admin_password", "security.allowed_origins", "security.allowed_methods",
		"security.allowed_headers",
	} {
		if err := v.BindEnv(key); err != nil {
			return nil, fmt.Errorf("bind environment variable for %s: %w", key, err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
