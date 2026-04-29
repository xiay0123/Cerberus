// Package handler 提供 Cerberus 服务的 HTTP 请求处理器。
//
// 该包包含两类处理器：
//   - AdminHandler: 管理接口（创建、查询、吊销、续期等）
//   - PublicHandler: 公开接口（激活、验证、心跳、换绑）
//
// 所有接口统一返回 JSON 格式：
//
//	{
//	    "code": 0,       // 0 表示成功
//	    "message": "",   // 错误信息
//	    "data": {...}    // 响应数据
//	}
package handler

import (
	"log"

	"cerberus.dev/server/internal/config"
	"cerberus.dev/server/internal/engine"
	"cerberus.dev/server/internal/middleware"
	"cerberus.dev/server/internal/response"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// loginReq 登录请求结构体。
type loginReq struct {
	// Token 管理员令牌。
	Token string `json:"token" binding:"required"`
}

// hLogin 处理管理员登录请求。
//
// 登录流程：
//  1. 验证管理员令牌
//  2. 生成 JWT Token
//  3. 返回 JWT 供后续请求使用
//
// 参数：
//   - cfg: 配置对象
//
// 返回：
//   - gin.HandlerFunc: Gin 处理函数
func hLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, 400, "token required")
			return
		}

		if req.Token != cfg.Auth.AdminToken {
			response.Error(c, 403, "invalid token")
			return
		}

		jwt, err := middleware.GenerateJWT(cfg.Auth.JWTSecret, "admin", cfg.Auth.JWTTTL)
		if err != nil {
			response.Error(c, 500, "generate jwt failed")
			return
		}

		response.OK(c, gin.H{"token": jwt})
	}
}

// SetupRouter 配置并返回 Gin 路由引擎。
//
// 路由配置：
//   - GET /health: 健康检查
//   - POST /auth/login: 管理员登录
//   - POST /api/v1/licenses/*: 管理接口（需要认证）
//   - POST /api/v1/activate: 激活机器（公开）
//   - POST /api/v1/verify: 验证 License（公开）
//   - POST /api/v1/heartbeat: 心跳上报（公开）
//   - POST /api/v1/unbind: 自助换绑（公开）
//
// 中间件配置：
//   - Recovery: 异常恢复
//   - CORS: 跨域支持
//   - RateLimiter: 限流（可选）
//   - Logger: 请求日志
//   - AdminOrJWTAuth: 管理接口认证
//
// 参数：
//   - cfg: 配置对象
//   - adminEngine: 管理引擎
//   - onlineEngine: 在线验证引擎
//   - db: 数据库连接
//
// 返回：
//   - *gin.Engine: Gin 引擎实例
func SetupRouter(cfg *config.Config, adminEngine *engine.AdminEngine, onlineEngine *engine.OnlineEngine, db *gorm.DB) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// CORS 配置
	corsOrigins := cfg.CORSOrigins
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"*"}
	}
	if len(corsOrigins) == 1 && corsOrigins[0] == "*" {
		log.Println("[WARN] CORS allows all origins (*), not recommended for production")
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 限流中间件
	if cfg.RateLimit.Enabled {
		r.Use(middleware.RateLimiter(cfg.RateLimit.RPS, cfg.RateLimit.Burst))
	}

	// 日志中间件
	r.Use(middleware.Logger())

	// 创建处理器
	adminHandler := NewAdminHandler(adminEngine, onlineEngine, db)
	publicHandler := NewPublicHandler(onlineEngine, adminEngine)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 登录
	r.POST("/auth/login", hLogin(cfg))

	// 管理接口（需要认证）
	api := r.Group("/api/v1")
	api.Use(middleware.AdminOrJWTAuth(cfg.Auth.AdminToken, cfg.Auth.JWTSecret))
	{
		licenses := api.Group("/licenses")
		{
			licenses.POST("/create", adminHandler.Create)
			licenses.POST("/get", adminHandler.Get)
			licenses.POST("/list", adminHandler.List)
			licenses.POST("/delete", adminHandler.Delete)
			licenses.POST("/revoke", adminHandler.Revoke)
			licenses.POST("/renew", adminHandler.Renew)
			licenses.POST("/unsuspend", adminHandler.Unsuspend)
			licenses.POST("/audit", adminHandler.AuditLogs)
			licenses.POST("/machines/revoke", adminHandler.RevokeMachine)
		}
	}

	// 公开接口
	r.POST("/api/v1/activate", publicHandler.Activate)
	r.POST("/api/v1/verify", publicHandler.Verify)
	r.POST("/api/v1/heartbeat", publicHandler.Heartbeat)
	r.POST("/api/v1/unbind", publicHandler.UnbindMachine)

	// 静态文件服务 (管理界面)
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
	r.Static("/assets", "./web/assets")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/index.html")
	})

	return r
}
