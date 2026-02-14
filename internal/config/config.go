package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Security   SecurityConfig   `mapstructure:"security"`
	Validation ValidationConfig `mapstructure:"validation"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type DatabaseConfig struct {
	Driver        string `mapstructure:"driver"`
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Name          string `mapstructure:"name"`
	User          string `mapstructure:"user"`
	Password      string `mapstructure:"password"`
	MaxOpenConns  int    `mapstructure:"max_open_conns"`
	MaxIdleConns  int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

type JWTConfig struct {
	Secret string        `mapstructure:"secret"`
	Expiry time.Duration `mapstructure:"expiry"`
	Issuer string        `mapstructure:"issuer"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	CacheTTL time.Duration `mapstructure:"cache_ttl"`
}

func (r *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type SecurityConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	Whitelist  []string `mapstructure:"whitelist"`
	Blacklist  []string `mapstructure:"blacklist"`
	RequireAuth bool    `mapstructure:"require_auth"`
}

func (s *SecurityConfig) IsTableAllowed(table string) bool {
	for _, blocked := range s.Blacklist {
		if strings.EqualFold(blocked, table) {
			return false
		}
	}

	if len(s.Whitelist) == 0 {
		return true
	}

	for _, allowed := range s.Whitelist {
		if strings.EqualFold(allowed, table) {
			return true
		}
	}

	return false
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type ValidationConfig struct {
	Enabled    bool                              `mapstructure:"enabled"`
	StrictMode bool                              `mapstructure:"strict_mode"`
	Rules      map[string]map[string][]RuleItem `mapstructure:"rules"`
}

type RuleItem struct {
	Type    string      `mapstructure:"type"`
	Pattern string      `mapstructure:"pattern"`
	Value   interface{} `mapstructure:"value"`
	Values  []string    `mapstructure:"values"`
	Message string      `mapstructure:"message"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
		v.ReadInConfig()
	}

	v.SetEnvPrefix("INSTANTGATE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	return nil
}
