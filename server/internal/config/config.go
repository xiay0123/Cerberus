// Package config 提供基于 Viper 的 YAML 配置加载功能。
//
// 支持从 config.yaml 读取服务端口、数据库路径、认证令牌、限流参数、GeoIP 配置等，
// 未在配置文件中指定的字段会使用合理的默认值。
//
// 配置文件示例：
//
//	server:
//	  port: 8080
//	  mode: release
//	database:
//	  path: ./data/cerberus.db
//	license:
//	  max_machines: 3
//	  heartbeat_ttl: 5m
//	auth:
//	  admin_token: your-secure-token
//	  jwt_secret: your-jwt-secret
//	  jwt_ttl: 24h
//	geoip:
//	  enabled: true
//	  policy: alert
//	  allowed_distance: 500
//	rate_limit:
//	  enabled: true
//	  rps: 20
//	  burst: 40
package config

import (
	"fmt"
	"time"

	"cerberus.dev/pkg/geoip"

	"github.com/spf13/viper"
)

// Config 是 Cerberus 服务的顶层配置结构，包含多个子模块的配置。
type Config struct {
	// Server HTTP 服务配置。
	Server ServerConfig `mapstructure:"server"`
	// Database 数据库配置。
	Database DatabaseConfig `mapstructure:"database"`
	// License License 相关配置。
	License LicenseConfig `mapstructure:"license"`
	// Auth 认证配置。
	Auth AuthConfig `mapstructure:"auth"`
	// RateLimit 限流配置。
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	// GeoIP 地理位置检测配置。
	GeoIP GeoIPConfig `mapstructure:"geoip"`
	// CORSOrigins CORS 允许的源。
	CORSOrigins []string `mapstructure:"cors_origins"`
}

// ServerConfig 定义 HTTP 服务的端口、运行模式及超时时间。
type ServerConfig struct {
	// Port 监听端口，默认 8080。
	Port int `mapstructure:"port"`
	// Mode Gin 运行模式：debug / release。
	Mode string `mapstructure:"mode"`
	// ReadTimeout 读取超时。
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	// WriteTimeout 写入超时。
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DatabaseConfig 定义 SQLite 数据库文件的存储路径。
type DatabaseConfig struct {
	// Path 数据库文件路径，默认 ./data/cerberus.db。
	Path string `mapstructure:"path"`
}

// LicenseConfig 定义机器绑定上限和心跳超时等授权相关配置。
type LicenseConfig struct {
	// MaxMachines 单 License 默认最大绑定机器数。
	MaxMachines int `mapstructure:"max_machines"`
	// HeartbeatTTL 心跳超时时间，超过后视为离线。
	HeartbeatTTL time.Duration `mapstructure:"heartbeat_ttl"`
}

// AuthConfig 定义管理端认证令牌和 JWT 签发参数。
type AuthConfig struct {
	// AdminToken 管理 API 的认证令牌。
	AdminToken string `mapstructure:"admin_token"`
	// JWTSecret JWT 签名密钥 (HS256)。
	JWTSecret string `mapstructure:"jwt_secret"`
	// JWTTTL JWT 有效期。
	JWTTTL time.Duration `mapstructure:"jwt_ttl"`
}

// RateLimitConfig 定义基于令牌桶算法的请求限流参数。
type RateLimitConfig struct {
	// Enabled 是否启用限流。
	Enabled bool `mapstructure:"enabled"`
	// RPS 每秒允许的请求数。
	RPS int `mapstructure:"rps"`
	// Burst 突发请求数上限。
	Burst int `mapstructure:"burst"`
}

// GeoIPConfig 定义地理位置检测相关参数。
type GeoIPConfig struct {
	// Enabled 是否启用地理位置检测。
	Enabled bool `mapstructure:"enabled"`
	// Policy 检测策略：allow / alert / deny。
	Policy geoip.Policy `mapstructure:"policy"`
	// AllowedDistance 允许的最大距离（公里）。
	AllowedDistance float64 `mapstructure:"allowed_distance"`
	// DatabasePath GeoIP 数据库路径（可选）。
	DatabasePath string `mapstructure:"database_path"`
}

// Load 从指定路径读取 YAML 配置文件，填充默认值后反序列化为 Config 结构体。
// 同时支持环境变量覆盖（Viper AutomaticEnv）。
//
// 参数：
//   - path: 配置文件路径
//
// 返回：
//   - *Config: 配置对象
//   - error: 加载失败时返回错误
//
// 示例：
//
//	cfg, err := config.Load("config.yaml")
func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// setDefaults 为所有配置项设置合理的默认值，确保未在 YAML 中配置时服务也能正常启动。
func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("server.read_timeout", "10s")
	viper.SetDefault("server.write_timeout", "10s")
	viper.SetDefault("database.path", "./data/cerberus.db")
	viper.SetDefault("license.max_machines", 3)
	viper.SetDefault("license.heartbeat_ttl", "5m")
	viper.SetDefault("auth.admin_token", "cerberus-admin-token-change-me")
	viper.SetDefault("auth.jwt_secret", "cerberus-jwt-secret-change-me")
	viper.SetDefault("auth.jwt_ttl", "24h")
	viper.SetDefault("rate_limit.enabled", true)
	viper.SetDefault("rate_limit.rps", 20)
	viper.SetDefault("rate_limit.burst", 40)
	viper.SetDefault("geoip.enabled", false)
	viper.SetDefault("geoip.policy", "allow")
	viper.SetDefault("geoip.allowed_distance", 500)
	viper.SetDefault("cors_origins", []string{"*"})
}
