// Package handler 提供 License 相关的 HTTP 请求处理器。
package handler

import (
	"fmt"

	"cerberus.dev/server/internal/engine"
	"cerberus.dev/server/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminHandler 管理接口处理器。
//
// AdminHandler 处理需要管理员权限的 API，
// 主要用于 License 的创建、查询、吊销、续期等管理操作。
type AdminHandler struct {
	// adminEngine 管理引擎。
	adminEngine *engine.AdminEngine
	// onlineEngine 在线验证引擎。
	onlineEngine *engine.OnlineEngine
	// db 数据库连接。
	db *gorm.DB
}

// NewAdminHandler 创建管理接口处理器。
//
// 参数：
//   - admin: 管理引擎
//   - online: 在线验证引擎
//   - db: 数据库连接
//
// 返回：
//   - *AdminHandler: 处理器实例
func NewAdminHandler(admin *engine.AdminEngine, online *engine.OnlineEngine, db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		adminEngine:  admin,
		onlineEngine: online,
		db:           db,
	}
}

// ============================================================
// 创建 License
// ============================================================

// createLicenseReq 创建 License 请求结构体。
type createLicenseReq struct {
	// Name 许可证名称。
	Name string `json:"name" binding:"required"`
	// Product 产品标识。
	Product string `json:"product" binding:"required"`
	// Issuer 发行者。
	Issuer string `json:"issuer"`
	// MaxMachines 最大绑定机器数。
	MaxMachines int `json:"max_machines"`
	// DurationSec 有效时长（秒）。
	DurationSec int64 `json:"duration_sec" binding:"required"`
	// MaxUnbindCount 最大换绑次数。
	MaxUnbindCount int `json:"max_unbind_count"`
	// IPBindingEnabled 是否启用 IP 绑定。
	IPBindingEnabled bool `json:"ip_binding_enabled"`
}

// Create 处理创建 License 请求。
//
// 请求：
//
//	POST /api/v1/licenses/create
//	{
//	    "name": "My License",
//	    "product": "MyApp",
//	    "duration_sec": 31536000,
//	    "max_machines": 3
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": {
//	        "id": "xxx-xxx-xxx",
//	        "name": "My License",
//	        ...
//	    }
//	}
func (h *AdminHandler) Create(c *gin.Context) {
	var req createLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	if req.DurationSec <= 0 {
		response.Error(c, 400, "duration_sec must be positive")
		return
	}

	l, err := h.adminEngine.Create(engine.CreateParams{
		Name:             req.Name,
		Product:          req.Product,
		Issuer:           req.Issuer,
		MaxMachines:      req.MaxMachines,
		DurationSec:      req.DurationSec,
		MaxUnbindCount:   req.MaxUnbindCount,
		IPBindingEnabled: req.IPBindingEnabled,
	})
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	h.adminEngine.AuditLog(l.ID, "create", fmt.Sprintf("license created, max_machines=%d, ip_binding=%v", l.MaxMachines, req.IPBindingEnabled), c.ClientIP())
	response.Created(c, l)
}

// ============================================================
// 查询 License
// ============================================================

// getLicenseReq 获取 License 请求结构体。
type getLicenseReq struct {
	// ID License ID。
	ID string `json:"id" binding:"required"`
}

// Get 处理获取 License 详情请求。
//
// 请求：
//
//	POST /api/v1/licenses/get
//	{
//	    "id": "xxx-xxx-xxx"
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": {
//	        "id": "xxx-xxx-xxx",
//	        "name": "My License",
//	        "machines": [...]
//	    }
//	}
func (h *AdminHandler) Get(c *gin.Context) {
	var req getLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	l, err := h.adminEngine.Get(req.ID)
	if err != nil {
		response.Error(c, 404, err.Error())
		return
	}

	response.OK(c, l)
}

// listLicenseReq 列表查询请求结构体。
type listLicenseReq struct {
	// Page 页码。
	Page int `json:"page"`
	// Size 每页数量。
	Size int `json:"size"`
	// Status 状态筛选。
	Status string `json:"status"`
	// Product 产品筛选。
	Product string `json:"product"`
}

// List 处理查询 License 列表请求。
//
// 请求：
//
//	POST /api/v1/licenses/list
//	{
//	    "page": 1,
//	    "size": 20,
//	    "status": "active"
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": {
//	        "items": [...],
//	        "total": 100,
//	        "page": 1,
//	        "size": 20
//	    }
//	}
func (h *AdminHandler) List(c *gin.Context) {
	var req listLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	result, err := h.adminEngine.List(engine.ListParams{
		Page:    req.Page,
		Size:    req.Size,
		Status:  req.Status,
		Product: req.Product,
	})
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	response.OK(c, gin.H{
		"items": result.Items,
		"total": result.Total,
		"page":  req.Page,
		"size":  req.Size,
	})
}

// ============================================================
// 吊销 License
// ============================================================

// revokeLicenseReq 吊销 License 请求结构体。
type revokeLicenseReq struct {
	// ID License ID。
	ID string `json:"id" binding:"required"`
}

// Revoke 处理吊销 License 请求。
//
// License 及其所有机器都将被标记为已吊销。
//
// 请求：
//
//	POST /api/v1/licenses/revoke
//	{
//	    "id": "xxx-xxx-xxx"
//	}
func (h *AdminHandler) Revoke(c *gin.Context) {
	var req revokeLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	if err := h.adminEngine.Revoke(req.ID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	h.adminEngine.AuditLog(req.ID, "revoke", "license revoked", c.ClientIP())
	response.OK(c, nil)
}

// reactivateLicenseReq 重新启用 License 请求结构体。
type reactivateLicenseReq struct {
	// ID License ID。
	ID string `json:"id" binding:"required"`
}

// Reactivate 处理重新启用已吊销 License 请求。
//
// 将 License 恢复为 active，并恢复其关联的已吊销机器。
//
// 请求：
//
//	POST /api/v1/licenses/reactivate
//	{
//	    "id": "xxx-xxx-xxx"
//	}
func (h *AdminHandler) Reactivate(c *gin.Context) {
	var req reactivateLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	if err := h.adminEngine.Reactivate(req.ID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	h.adminEngine.AuditLog(req.ID, "reactivate", "license reactivated", c.ClientIP())
	response.OK(c, gin.H{"message": "license reactivated successfully"})
}

// ============================================================
// 续期 License
// ============================================================

// renewLicenseReq 续期 License 请求结构体。
type renewLicenseReq struct {
	// ID License ID。
	ID string `json:"id" binding:"required"`
	// DurationSec 续期时长（秒）。
	DurationSec int64 `json:"duration_sec" binding:"required"`
}

// Renew 处理续期 License 请求。
//
// 请求：
//
//	POST /api/v1/licenses/renew
//	{
//	    "id": "xxx-xxx-xxx",
//	    "duration_sec": 31536000
//	}
func (h *AdminHandler) Renew(c *gin.Context) {
	var req renewLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	if req.DurationSec <= 0 {
		response.Error(c, 400, "duration_sec must be positive")
		return
	}

	if err := h.adminEngine.Renew(req.ID, req.DurationSec); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	h.adminEngine.AuditLog(req.ID, "renew", fmt.Sprintf("license renewed by %d seconds", req.DurationSec), c.ClientIP())
	response.OK(c, gin.H{"message": "license renewed successfully"})
}

// ============================================================
// 解封 License
// ============================================================

// unsuspendLicenseReq 解封 License 请求结构体。
type unsuspendLicenseReq struct {
	// ID License ID。
	ID string `json:"id" binding:"required"`
	// ResetUnbindCount 是否重置换绑计数。
	ResetUnbindCount bool `json:"reset_unbind_count"`
}

// Unsuspend 处理解封 License 请求。
//
// 当 License 因换绑次数超限被暂停时，管理员可以执行解封。
//
// 请求：
//
//	POST /api/v1/licenses/unsuspend
//	{
//	    "id": "xxx-xxx-xxx",
//	    "reset_unbind_count": true
//	}
func (h *AdminHandler) Unsuspend(c *gin.Context) {
	var req unsuspendLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	if err := h.adminEngine.Unsuspend(req.ID, req.ResetUnbindCount); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	h.adminEngine.AuditLog(req.ID, "unsuspend", fmt.Sprintf("license unsuspended, reset_unbind_count=%v", req.ResetUnbindCount), c.ClientIP())
	response.OK(c, gin.H{"message": "license unsuspended successfully"})
}

// ============================================================
// 删除 License
// ============================================================

// deleteLicenseReq 删除 License 请求结构体。
type deleteLicenseReq struct {
	// ID License ID。
	ID string `json:"id" binding:"required"`
}

// Delete 处理删除 License 请求。
//
// 执行软删除，数据不会真正删除，只是标记为已删除。
//
// 请求：
//
//	POST /api/v1/licenses/delete
//	{
//	    "id": "xxx-xxx-xxx"
//	}
func (h *AdminHandler) Delete(c *gin.Context) {
	var req deleteLicenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	if err := h.adminEngine.Delete(req.ID); err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	h.adminEngine.AuditLog(req.ID, "delete", "license deleted", c.ClientIP())
	response.OK(c, nil)
}

// ============================================================
// 审计日志
// ============================================================

// auditLogsReq 审计日志请求结构体。
type auditLogsReq struct {
	// ID License ID。
	ID string `json:"id" binding:"required"`
}

// AuditLogs 处理获取审计日志请求。
//
// 请求：
//
//	POST /api/v1/licenses/audit
//	{
//	    "id": "xxx-xxx-xxx"
//	}
//
// 响应：
//
//	{
//	    "code": 0,
//	    "data": [
//	        {"action": "create", "detail": "...", "created_at": "..."},
//	        ...
//	    ]
//	}
func (h *AdminHandler) AuditLogs(c *gin.Context) {
	var req auditLogsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	logs, err := h.adminEngine.GetAuditLogs(req.ID)
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	response.OK(c, logs)
}

// ============================================================
// 吊销机器
// ============================================================

// revokeMachineReq 吊销机器请求结构体。
type revokeMachineReq struct {
	// LicenseID License ID。
	LicenseID string `json:"license_id" binding:"required"`
	// MachineID 机器 ID。
	MachineID string `json:"machine_id" binding:"required"`
}

// RevokeMachine 处理吊销机器请求。
//
// 管理员可以主动吊销单个机器绑定，让用户能够绑定新机器。
//
// 请求：
//
//	POST /api/v1/licenses/machines/revoke
//	{
//	    "license_id": "xxx-xxx-xxx",
//	    "machine_id": "yyy-yyy-yyy"
//	}
func (h *AdminHandler) RevokeMachine(c *gin.Context) {
	var req revokeMachineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	if err := h.adminEngine.RevokeMachine(req.LicenseID, req.MachineID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	h.adminEngine.AuditLog(req.LicenseID, "revoke_machine", "machine revoked: "+req.MachineID, c.ClientIP())
	response.OK(c, gin.H{"message": "machine revoked, you can now activate a new device"})
}
