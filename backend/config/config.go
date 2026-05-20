// Package config 负责从环境变量（或可选的 config.yaml）加载应用配置。
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 汇聚所有子配置。
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Security SecurityConfig
}

// ServerConfig 包含 HTTP 服务器配置。
type ServerConfig struct {
	Port string
	Env  string // development | production
}

// DatabaseConfig 包含 PostgreSQL 连接参数。
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN 返回 GORM 所需的 PostgreSQL Data Source Name。
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// RedisConfig 包含 Redis 连接参数。
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// SecurityConfig 包含安全相关参数。
type SecurityConfig struct {
	// MasterKey 是 AES-256-GCM 主密钥，必须恰好 32 字节，通过环境变量 MASTER_KEY 注入。
	MasterKey string
	// AdminPassword 是后台管理界面的登录密码，通过环境变量 ADMIN_PASSWORD 注入。
	AdminPassword string
}

// Load 从环境变量读取配置，并尝试加载可选的 config/config.yaml（不存在则忽略）。
func Load() (*Config, error) {
	v := viper.New()

	// 默认值
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.env", "production")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "paygate")
	v.SetDefault("database.name", "paygate_omni")
	v.SetDefault("database.sslmode", "require")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)

	// 环境变量绑定（点号转下划线，全部大写）
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	bindings := map[string]string{
		"server.port":          "SERVER_PORT",
		"server.env":           "APP_ENV",
		"database.host":        "DB_HOST",
		"database.port":        "DB_PORT",
		"database.user":        "DB_USER",
		"database.password":    "DB_PASSWORD",
		"database.name":        "DB_NAME",
		"database.sslmode":     "DB_SSLMODE",
		"redis.addr":           "REDIS_ADDR",
		"redis.password":       "REDIS_PASSWORD",
		"redis.db":             "REDIS_DB",
		"security.masterkey":   "MASTER_KEY",
		"security.adminpassword": "ADMIN_PASSWORD",
	}
	for key, env := range bindings {
		if err := v.BindEnv(key, env); err != nil {
			return nil, fmt.Errorf("config: bind env %s: %w", env, err)
		}
	}

	// 可选 YAML 文件
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig()

	// MASTER_KEY 强制校验
	masterKey := v.GetString("security.masterkey")
	if masterKey == "" {
		return nil, fmt.Errorf("config: MASTER_KEY environment variable is required")
	}
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("config: MASTER_KEY must be exactly 32 bytes, got %d", len(masterKey))
	}

	// ADMIN_PASSWORD 强制校验
	adminPassword := v.GetString("security.adminpassword")
	if adminPassword == "" {
		return nil, fmt.Errorf("config: ADMIN_PASSWORD environment variable is required")
	}

	return &Config{
		Server: ServerConfig{
			Port: v.GetString("server.port"),
			Env:  v.GetString("server.env"),
		},
		Database: DatabaseConfig{
			Host:     v.GetString("database.host"),
			Port:     v.GetInt("database.port"),
			User:     v.GetString("database.user"),
			Password: v.GetString("database.password"),
			Name:     v.GetString("database.name"),
			SSLMode:  v.GetString("database.sslmode"),
		},
		Redis: RedisConfig{
			Addr:     v.GetString("redis.addr"),
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		Security: SecurityConfig{
			MasterKey:     masterKey,
			AdminPassword: adminPassword,
		},
	}, nil
}
