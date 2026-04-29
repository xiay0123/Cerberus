// Package middleware 提供 Cerberus 服务的 HTTP 中间件。
//
// 本包包含以下中间件：
//   - AdminAuth: Admin Token 认证，保护管理接口
//   - JWTAuth: JWT Token 认证
//   - AdminOrJWTAuth: Admin Token 或 JWT 双认证
//   - RateLimiter: per-IP 请求限流，防止暴力攻击
//   - Logger: 请求日志记录
//
// 中间件执行顺序（推荐）：
//
//	Recovery → CORS → RateLimiter → Logger → Auth → Handler
package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"cerberus.dev/server/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

// AdminAuth 创建 Admin Token 认证中间件。
//
// 验证请求头中的 Authorization 是否包含有效的 Admin Token。
// Token 可以通过两种方式传递：
//   - 请求头：Authorization: Bearer <token>
//   - 查询参数：?token=<token>
//
// 参数：
//   - adminToken: 配置的 Admin Token 值
//
// 返回：
//   - gin.HandlerFunc: Gin 中间件函数
//
// 使用示例：
//
//	api := r.Group("/api/v1")
//	api.Use(middleware.AdminAuth(cfg.Auth.AdminToken))
func AdminAuth(adminToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求中提取 Token
		token := extractToken(c)

		// 未提供 Token
		if token == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// Token 不匹配
		if token != adminToken {
			response.Error(c, http.StatusForbidden, "invalid admin token")
			c.Abort()
			return
		}

		// 验证通过，继续执行后续处理器
		c.Next()
	}
}

// JWTAuth 创建 JWT Token 认证中间件。
//
// 解析并验证 JWT Token，支持 HS256 签名算法。
// 验证通过后，将 JWT Claims 存入 Gin 上下文，供后续处理器使用。
//
// 参数：
//   - secret: JWT 签名密钥
//
// 返回：
//   - gin.HandlerFunc: Gin 中间件函数
//
// 使用示例：
//
//	api := r.Group("/api/v1")
//	api.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求中提取 Token
		tokenStr := extractToken(c)

		// 未提供 Token
		if tokenStr == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// 解析并验证 JWT
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// 验证签名算法是否为 HS256
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		// Token 无效或已过期
		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		// 将 Claims 存入上下文，供后续使用
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("jwt_claims", claims)
		}

		c.Next()
	}
}

// AdminOrJWTAuth 创建 Admin Token 或 JWT 双认证中间件。
//
// 支持 Admin Token（直接认证）或 JWT Token（登录后认证）两种方式。
// 适用于管理接口，允许通过 Admin Token 快速操作或通过 JWT 会话操作。
func AdminOrJWTAuth(adminToken, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// 先尝试 Admin Token
		if token == adminToken {
			c.Next()
			return
		}

		// 再尝试 JWT
		jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !jwtToken.Valid {
			response.Error(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
			c.Set("jwt_claims", claims)
		}

		c.Next()
	}
}

// GenerateJWT 生成 JWT Token。
//
// 使用 HS256 算法签名，包含标准 Claims（sub, iat, exp）。
//
// 参数：
//   - secret: 签名密钥
//   - subject: Token 主题（通常是用户标识）
//   - ttl: Token 有效期
//
// 返回：
//   - string: JWT Token 字符串
//   - error: 生成失败时返回错误
//
// 使用示例：
//
//	token, err := middleware.GenerateJWT(secret, "admin", 24*time.Hour)
func GenerateJWT(secret, subject string, ttl time.Duration) (string, error) {
	now := time.Now()

	// 构建标准 Claims
	claims := jwt.RegisteredClaims{
		Subject:   subject,                          // 主题
		IssuedAt:  jwt.NewNumericDate(now),          // 签发时间
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), // 过期时间
	}

	// 创建 Token 并签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ============================================================
// Per-IP Rate Limiter
// ============================================================

// ipLimiter 存储 per-IP 限流器
type ipLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// newIPLimiter 创建 per-IP 限流器管理器
func newIPLimiter(rps, burst int) *ipLimiter {
	return &ipLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// getLimiter 获取或创建指定 IP 的限流器
func (l *ipLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.limiters[ip]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 双重检查
	if limiter, exists = l.limiters[ip]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(l.rps, l.burst)
	l.limiters[ip] = limiter
	return limiter
}

// cleanup 定期清理长时间未使用的限流器（可选优化）
func (l *ipLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	// 简单实现：清空所有限流器，让它们按需重建
	// 生产环境可以用 LRU 缓存策略
	l.limiters = make(map[string]*rate.Limiter)
}

// globalIPLimiter 全局 per-IP 限流器
var globalIPLimiter *ipLimiter

// RateLimiter 创建 per-IP 请求限流中间件。
//
// 基于令牌桶算法实现，为每个 IP 地址独立限流，防止暴力攻击。
//
// 参数：
//   - rps: 每秒允许的请求数（令牌放入速率）
//   - burst: 突发请求数上限（桶容量）
//
// 返回：
//   - gin.HandlerFunc: Gin 中间件函数
//
// 使用示例：
//
//	r.Use(middleware.RateLimiter(20, 40))  // 20 RPS，最大突发 40
func RateLimiter(rps, burst int) gin.HandlerFunc {
	globalIPLimiter = newIPLimiter(rps, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := globalIPLimiter.getLimiter(ip)

		if !limiter.Allow() {
			response.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Logger 创建请求日志中间件。
//
// 记录每个请求的处理耗时和基本信息。
//
// 返回：
//   - gin.HandlerFunc: Gin 中间件函数
//
// 使用示例：
//
//	r.Use(middleware.Logger())
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		// 记录请求日志：方法、路径、状态码、耗时
		log.Printf("[%s] %s %d %v",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			latency,
		)
	}
}

// extractToken 从请求中提取认证 Token。
//
// 支持两种方式：
//  1. 请求头 Authorization: Bearer <token>
//  2. 查询参数 ?token=<token>
//
// 参数：
//   - c: Gin 上下文
//
// 返回：
//   - string: 提取的 Token，未找到返回空字符串
func extractToken(c *gin.Context) string {
	// 尝试从 Authorization 头获取
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// 尝试从查询参数获取
	return c.Query("token")
}
