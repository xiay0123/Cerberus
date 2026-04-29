// Package main 是 Cerberus License 验证服务的入口点。
//
// Cerberus 是一个轻量级的软件授权管理系统，支持：
//   - 在线 License 验证
//   - 机器绑定与换绑
//   - 地理位置检测（异地登录告警）
//   - 心跳监控
//   - 审计日志
//
// 启动方式：
//
//	cerberus-server
//
// 配置文件（config.yaml）示例：
//
//	server:
//	  port: 8080
//	auth:
//	  admin_token: your-secure-token
package main

import (
	"fmt"
	"log"

	"cerberus.dev/pkg/geoip"
	"cerberus.dev/server/internal/config"
	"cerberus.dev/server/internal/database"
	"cerberus.dev/server/internal/engine"
	"cerberus.dev/server/internal/handler"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 初始化数据库
	db, err := database.Init(cfg.Database.Path)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}

	// 创建引擎
	adminEngine := engine.NewAdminEngine(db, cfg.License.MaxMachines, cfg.License.HeartbeatTTL)
	onlineEngine := engine.NewOnlineEngine(db, cfg.License.MaxMachines, cfg.License.HeartbeatTTL)

	// 初始化 GeoIP 检测器
	if cfg.GeoIP.Enabled {
		detector, err := geoip.NewDetector(geoip.Config{
			Policy:          cfg.GeoIP.Policy,
			AllowedDistance: cfg.GeoIP.AllowedDistance,
			DatabasePath:    cfg.GeoIP.DatabasePath,
			Enabled:         cfg.GeoIP.Enabled,
		})
		if err != nil {
			log.Printf("WARN: failed to initialize GeoIP detector: %v", err)
		} else {
			onlineEngine.SetGeoIPDetector(detector)
			log.Printf("GeoIP detector enabled with policy: %s", cfg.GeoIP.Policy)
		}
	}

	// 设置路由
	r := handler.SetupRouter(cfg, adminEngine, onlineEngine, db)

	// 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Cerberus starting on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
