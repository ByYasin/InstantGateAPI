package config

import (
	"time"

	"github.com/spf13/viper"
)

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)

	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.name", "instantgate")
	v.SetDefault("database.user", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expiry", 24*time.Hour)
	v.SetDefault("jwt.issuer", "instantgate")

	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.cache_ttl", 5*time.Minute)

	v.SetDefault("security.enabled", true)
	v.SetDefault("security.require_auth", false)
	v.SetDefault("security.whitelist", []string{})
	v.SetDefault("security.blacklist", []string{})

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	v.SetDefault("validation.enabled", true)
	v.SetDefault("validation.strict_mode", false)
	v.SetDefault("validation.rules", map[string]interface{}{})
}
