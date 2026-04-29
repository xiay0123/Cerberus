// Package handler 提供 License 相关的 HTTP 请求处理器。
//
// 该包包含两类处理器：
//   - PublicHandler: 公开接口（激活、验证、心跳、换绑）
//   - AdminHandler: 管理接口（创建、查询、吊销、续期等）
package handler

import (
	"fmt"
	"strconv"

	"cerberus.dev/server/internal/engine"
	"cerberus.dev/server/internal/response"

	"github.com/gin-gonic/gin"
)

// PublicHandler 公开接口处理器。
//
// PublicHandler 处理不需要管理员权限的公开 API，
// 主要用于客户端的激活、验证、心跳和换绑操作。
type PublicHandler struct {
	// onlineEngine 在线验证引擎。
	onlineEngine *engine.OnlineEngine
	// adminEngine 管理引擎（用于记录审计日志）。
	adminEngine *engine.AdminEngine
}

// NewPublicHandler 创建公开接口处理器。
//
// 参数：
//   - online: 在线验证引擎
//   - admin: 管理引擎
//
// 返回：
//   - *PublicHandler: 处理器实例
func NewPublicHandler(online *engine.OnlineEngine, admin *engine.AdminEngine) *PublicHandler {
	return &PublicHandler{
		onlineEngine: online,
		adminEngine:  admin,
	}
}

// ============================================================
// 机器激活
// ============================================================

// activateReq 激活请求结构体。
type activateReq struct {
	// LicenseID License ID。
	LicenseID string `json:"license_id" binding:"required"`
	// Fingerprint 机器指纹。
	Fingerprint string `json:"fingerprint" binding:"required"`
	// Hostname 主机名。
	Hostname string `json:"hostname"`
	// OS 操作系统。
	OS string `json:"os"`
	// Arch 系统架构。
	Arch string `json:"arch"`
	// IP IP 地址。
	IP string `json:"ip"`
}

// Activate 处理机器激活请求。
//
// 激活流程：
//  1. 验证请求参数
//  2. 调用在线引擎执行激活
//  3. 记录审计日志
//  4. 返回激活结果（包含地理位置告警信息）
//
// 请求：
//
//	POST /api/v1/activate
//	{
//	    "license_id": "xxx",
//	    "fingerprint": "xxx",
//	    "hostname": "MyPC",
//	    "os": "windows",
//	    "arch": "amd64"
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": {
//	        "machine": {...},
//	        "geoip_alert": "remote login detected (800 km away)"
//	    }
//	}
func (h *PublicHandler) Activate(c *gin.Context) {
	var req activateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	// IP 获取优先级：请求体 > X-Forwarded-For > X-Real-IP > ClientIP
	ip := req.IP
	if ip == "" {
		ip = clientIP(c, "")
	}

	result, err := h.onlineEngine.Activate(req.LicenseID, engine.ActivateParams{
		Fingerprint: req.Fingerprint,
		Hostname:    req.Hostname,
		OS:          req.OS,
		Arch:        req.Arch,
		IP:          ip,
	})
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	// 记录审计日志
	action := "activate"
	detail := "machine activated: " + req.Fingerprint
	if result.GeoIPAlert != "" {
		detail += " | geoip: " + result.GeoIPAlert
		action = "activate_geoip_alert"
	}
	h.adminEngine.AuditLog(req.LicenseID, action, detail, c.ClientIP())

	response.OK(c, gin.H{
		"machine":        result.Machine,
		"geoip_alert":    result.GeoIPAlert,
		"is_new_machine": result.IsNewMachine,
	})
}

// ============================================================
// 验证 License
// ============================================================

// verifyReq 验证请求结构体。
type verifyReq struct {
	// LicenseID License ID。
	LicenseID string `json:"license_id" binding:"required"`
	// Fingerprint 机器指纹（可选）。
	Fingerprint string `json:"fingerprint"`
	// IP IP 地址（可选）。
	IP string `json:"ip"`
}

// Verify 处理验证请求。
//
// 验证流程：
//  1. 验证请求参数
//  2. 调用在线引擎执行验证
//  3. 记录审计日志
//  4. 返回验证结果
//
// 请求：
//
//	POST /api/v1/verify
//	{
//	    "license_id": "xxx",
//	    "fingerprint": "xxx"
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": {
//	        "valid": true,
//	        "license_id": "xxx",
//	        "product": "MyApp",
//	        "expires_in": 86400
//	    }
//	}
func (h *PublicHandler) Verify(c *gin.Context) {
	var req verifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	// IP 获取优先级：请求体 > X-Forwarded-For > X-Real-IP > ClientIP
	ip := req.IP
	if ip == "" {
		ip = clientIP(c, "")
	}

	result := h.onlineEngine.Verify(req.LicenseID, req.Fingerprint, ip)
	h.adminEngine.AuditLog(req.LicenseID, "verify", "verify result: valid="+strconv.FormatBool(result.Valid), c.ClientIP())
	response.OK(c, result)
}

// ============================================================
// 心跳上报
// ============================================================

// heartbeatReq 心跳请求结构体。
type heartbeatReq struct {
	// LicenseID License ID。
	LicenseID string `json:"license_id" binding:"required"`
	// Fingerprint 机器指纹。
	Fingerprint string `json:"fingerprint" binding:"required"`
	// IP IP 地址。
	IP string `json:"ip"`
}

// Heartbeat 处理心跳上报请求。
//
// 心跳流程：
//  1. 验证请求参数
//  2. 调用在线引擎更新机器状态
//  3. 返回处理结果
//
// 请求：
//
//	POST /api/v1/heartbeat
//	{
//	    "license_id": "xxx",
//	    "fingerprint": "xxx"
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": null
//	}
func (h *PublicHandler) Heartbeat(c *gin.Context) {
	var req heartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	// IP 获取优先级：请求体 > X-Forwarded-For > X-Real-IP > ClientIP
	ip := req.IP
	if ip == "" {
		ip = clientIP(c, "")
	}

	if err := h.onlineEngine.Heartbeat(req.LicenseID, req.Fingerprint, ip); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.OK(c, nil)
}

// ============================================================
// 客户端自助换绑
// ============================================================

// unbindMachineReq 换绑请求结构体。
type unbindMachineReq struct {
	// LicenseID License ID。
	LicenseID string `json:"license_id" binding:"required"`
	// OldFingerprint 要解绑的机器指纹。
	OldFingerprint string `json:"old_fingerprint" binding:"required"`
}

// UnbindMachine 处理客户端自助换绑请求。
//
// 换绑流程：
//  1. 验证请求参数
//  2. 调用在线引擎执行换绑
//  3. 记录审计日志
//  4. 返回换绑结果
//
// 请求：
//
//	POST /api/v1/unbind
//	{
//	    "license_id": "xxx",
//	    "old_fingerprint": "xxx"
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": {
//	        "success": true,
//	        "remaining": 2,
//	        "message": "machine unbound successfully"
//	    }
//	}
func (h *PublicHandler) UnbindMachine(c *gin.Context) {
	var req unbindMachineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	result, err := h.onlineEngine.UnbindMachine(req.LicenseID, req.OldFingerprint)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	h.adminEngine.AuditLog(req.LicenseID, "unbind", fmt.Sprintf("machine unbound: %s, remaining: %d", req.OldFingerprint, result.Remaining), c.ClientIP())
	response.OK(c, result)
}

// ============================================================
// 辅助函数
// ============================================================

// clientIP 获取客户端 IP 地址。
//
// 优先使用 Gin 框架获取的 IP，如果获取失败则使用 fallback 值。
//
// 参数：
//   - c: Gin 上下文
//   - fallback: 备用 IP 地址
//
// 返回：
//   - string: 客户端 IP 地址
func clientIP(c *gin.Context, fallback string) string {
	// 尝试从 X-Forwarded-For 或 X-Real-IP 获取
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For 可能包含多个 IP，取第一个
		parts := splitIPs(ip)
		if len(parts) > 0 && parts[0] != "" {
			return trimPort(parts[0])
		}
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return trimPort(ip)
	}

	// 使用 Gin 的 ClientIP
	ip := c.ClientIP()
	if ip != "" {
		// 转换 IPv6 回环地址为 IPv4 格式
		if ip == "::1" {
			return "127.0.0.1"
		}
		return ip
	}

	// 使用 fallback
	if fallback != "" {
		return fallback
	}

	return ""
}

// splitIPs 分割 X-Forwarded-For 中的多个 IP
func splitIPs(s string) []string {
	var result []string
	for _, p := range splitByComma(s) {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, trim(s[start:i]))
			start = i + 1
		}
	}
	result = append(result, trim(s[start:]))
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// trimPort 移除 IP 中的端口部分
func trimPort(ip string) string {
	// IPv6 地址可能包含冒号，需要特殊处理
	if len(ip) > 0 && ip[0] == '[' {
		// IPv6 格式 [::1]:port
		for i := 0; i < len(ip); i++ {
			if ip[i] == ']' {
				if i+1 < len(ip) && ip[i+1] == ':' {
					return ip[:i+1]
				}
				return ip
			}
		}
	}
	// IPv4 格式
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
		if ip[i] == '.' || (ip[i] >= '0' && ip[i] <= '9') {
			continue
		}
		break
	}
	return ip
}
