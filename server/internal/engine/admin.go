// Package admin 提供 Cerberus 管理引擎。
//
// 该包实现 License 的管理功能：
//   - 创建 License
//   - 吊销、续期、解封 License
//   - 管理 License 列表
//   - 管理机器绑定
//   - 审计日志记录
package engine

import (
	"fmt"
	"time"

	"cerberus.dev/pkg/types"
	"cerberus.dev/server/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminEngine 管理引擎。
//
// AdminEngine 提供 License 的管理功能，包括：
//   - 创建、查询、删除 License
//   - 吊销、续期、解封 License
//   - 管理机器绑定
//   - 记录审计日志
type AdminEngine struct {
	// db 数据库连接。
	db *gorm.DB
	// maxMachines 默认最大机器数。
	maxMachines int
	// heartbeatTTL 心跳超时时间。
	heartbeatTTL time.Duration
}

// NewAdminEngine 创建管理引擎实例。
//
// 参数：
//   - db: GORM 数据库连接
//   - maxMachines: 默认最大绑定机器数（License 未指定时使用）
//   - heartbeatTTL: 心跳超时时间
//
// 返回：
//   - *AdminEngine: 引擎实例
//
// 示例：
//
//	engine := NewAdminEngine(db, 5, 24*time.Hour)
func NewAdminEngine(db *gorm.DB, maxMachines int, heartbeatTTL time.Duration) *AdminEngine {
	return &AdminEngine{
		db:            db,
		maxMachines:   maxMachines,
		heartbeatTTL:  heartbeatTTL,
	}
}

// CreateParams 创建 License 参数。
//
// 包含创建 License 所需的全部信息：
//   - Name: 许可证名称（显示用）
//   - Product: 产品标识（用于区分不同产品）
//   - Issuer: 发行者名称
//   - MaxMachines: 最大绑定机器数
//   - DurationSec: 有效时长（秒）
//   - MaxUnbindCount: 最大换绑次数
//   - IPBindingEnabled: 是否启用 IP 绑定
type CreateParams struct {
	// Name 许可证名称。
	Name string
	// Product 产品标识。
	Product string
	// Issuer 发行者。
	Issuer string
	// MaxMachines 最大绑定机器数。
	MaxMachines int
	// DurationSec 有效时长（秒）。
	DurationSec int64
	// MaxUnbindCount 最大换绑次数。
	MaxUnbindCount int
	// IPBindingEnabled 是否启用 IP 绑定。
	IPBindingEnabled bool
}

// Create 创建 License。
//
// 创建流程：
//  1. 验证参数（时长必须为正数）
//  2. 设置默认值（最大机器数、换绑次数）
//  3. 计算 ValidFrom 和 ValidUntil
//  4. 生成 UUID 并保存到数据库
//
// 参数：
//   - params: 创建参数
//
// 返回：
//   - *model.License: 创建的 License
//   - error: 创建失败错误
func (e *AdminEngine) Create(params CreateParams) (*model.License, error) {
	now := time.Now().Unix()

	if params.DurationSec <= 0 {
		return nil, fmt.Errorf("duration_sec must be positive")
	}

	if params.MaxMachines <= 0 {
		params.MaxMachines = e.maxMachines
	}

	l := &model.License{
		ID:               uuid.New().String(),
		Name:             params.Name,
		Product:          params.Product,
		Issuer:           params.Issuer,
		MaxMachines:      params.MaxMachines,
		IPBindingEnabled: params.IPBindingEnabled,
		ValidFrom:        now,
		ValidUntil:       now + params.DurationSec,
		DurationSec:      params.DurationSec,
		Status:           types.LicenseActive,
		MaxUnbindCount:   params.MaxUnbindCount,
	}

	if l.MaxUnbindCount <= 0 {
		l.MaxUnbindCount = 3
	}

	if err := e.db.Create(l).Error; err != nil {
		return nil, fmt.Errorf("save license: %w", err)
	}

	return l, nil
}

// Revoke 吊销 License。
//
// 吊销操作：
//  1. 将 License 状态设为 revoked
//  2. 将所有关联的机器状态设为 revoked
//
// 注意：吊销操作不可逆。
//
// 参数：
//   - licenseID: License ID
//
// 返回：
//   - error: 吊销失败错误
func (e *AdminEngine) Revoke(licenseID string) error {
	result := e.db.Model(&model.License{}).
		Where("id = ? AND status = ?", licenseID, types.LicenseActive).
		Update("status", types.LicenseRevoked)

	if result.RowsAffected == 0 {
		return fmt.Errorf("license not found or already revoked")
	}

	e.db.Model(&model.Machine{}).
		Where("license_id = ?", licenseID).
		Update("status", types.MachineRevoked)

	return nil
}

// Renew 续期 License。
//
// 续期规则：
//   - 如果 License 已过期，从当前时间开始计算新有效期
//   - 如果 License 未过期，在原有效期基础上延长
//   - 已吊销的 License 无法续期
//
// 参数：
//   - licenseID: License ID
//   - durationSec: 续期时长（秒）
//
// 返回：
//   - error: 续期失败错误
func (e *AdminEngine) Renew(licenseID string, durationSec int64) error {
	var l model.License
	if err := e.db.Where("id = ?", licenseID).First(&l).Error; err != nil {
		return fmt.Errorf("license not found")
	}

	if l.Status == types.LicenseRevoked {
		return fmt.Errorf("cannot renew revoked license")
	}

	newValidUntil := l.ValidUntil + durationSec
	if l.ValidUntil < time.Now().Unix() {
		newValidUntil = time.Now().Unix() + durationSec
	}

	return e.db.Model(&l).Updates(map[string]interface{}{
		"valid_until": newValidUntil,
		"status":      types.LicenseActive,
	}).Error
}

// Unsuspend 解封 License。
//
// 当 License 因换绑次数超限被暂停时，管理员可以执行解封操作。
// 可选择是否重置换绑计数。
//
// 参数：
//   - licenseID: License ID
//   - resetUnbindCount: 是否重置换绑计数
//
// 返回：
//   - error: 解封失败错误
func (e *AdminEngine) Unsuspend(licenseID string, resetUnbindCount bool) error {
	var l model.License
	if err := e.db.Where("id = ?", licenseID).First(&l).Error; err != nil {
		return fmt.Errorf("license not found")
	}

	if l.Status != types.LicenseSuspended {
		return fmt.Errorf("license is not suspended")
	}

	updates := map[string]interface{}{
		"status": types.LicenseActive,
	}
	if resetUnbindCount {
		updates["unbind_count"] = 0
	}

	return e.db.Model(&l).Updates(updates).Error
}

// RevokeMachine 吊销机器。
//
// 管理员可以主动吊销单个机器绑定。
// 吊销后会递增 License 的换绑计数。
//
// 参数：
//   - licenseID: License ID
//   - machineID: 机器 ID
//
// 返回：
//   - error: 吊销失败错误
func (e *AdminEngine) RevokeMachine(licenseID, machineID string) error {
	result := e.db.Model(&model.Machine{}).
		Where("id = ? AND license_id = ? AND status = ?", machineID, licenseID, types.MachineActive).
		Update("status", types.MachineRevoked)

	if result.RowsAffected == 0 {
		return fmt.Errorf("machine not found or already revoked")
	}

	e.db.Model(&model.License{}).Where("id = ?", licenseID).
		UpdateColumn("unbind_count", gorm.Expr("unbind_count + 1"))

	return nil
}

// Delete 软删除 License。
//
// 软删除不会真正删除数据，而是标记为已删除。
// 软删除的 License 不会出现在正常查询结果中。
//
// 参数：
//   - licenseID: License ID
//
// 返回：
//   - error: 删除失败错误
func (e *AdminEngine) Delete(licenseID string) error {
	return e.db.Where("id = ?", licenseID).Delete(&model.License{}).Error
}

// Get 获取 License 详情。
//
// 返回 License 信息及其关联的所有机器。
//
// 参数：
//   - licenseID: License ID
//
// 返回：
//   - *model.License: License 详情
//   - error: 查询失败错误
func (e *AdminEngine) Get(licenseID string) (*model.License, error) {
	var l model.License
	if err := e.db.Preload("Machines").Where("id = ?", licenseID).First(&l).Error; err != nil {
		return nil, fmt.Errorf("license not found")
	}
	return &l, nil
}

// ListResult 列表结果。
type ListResult struct {
	// Items License 列表。
	Items []model.License
	// Total 总数。
	Total int64
}

// ListParams 列表查询参数。
type ListParams struct {
	// Page 页码（从 1 开始）。
	Page int
	// Size 每页数量。
	Size int
	// Status 状态筛选（可选）。
	Status string
	// Product 产品筛选（可选）。
	Product string
}

// List 查询 License 列表。
//
// 支持分页和状态/产品筛选。
// 默认按创建时间倒序排列。
//
// 参数：
//   - params: 查询参数
//
// 返回：
//   - *ListResult: 列表结果
//   - error: 查询失败错误
func (e *AdminEngine) List(params ListParams) (*ListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}
	if params.Size > 100 {
		params.Size = 100
	}

	var licenses []model.License
	query := e.db.Order("created_at DESC")

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Product != "" {
		query = query.Where("product = ?", params.Product)
	}

	var total int64
	query.Model(&model.License{}).Count(&total)
	query.Offset((params.Page - 1) * params.Size).Limit(params.Size).Find(&licenses)

	return &ListResult{
		Items: licenses,
		Total: total,
	}, nil
}

// AuditLog 记录审计日志。
//
// 审计日志用于追踪所有关键操作，包括：
//   - 客户端操作：激活、验证、心跳、换绑
//   - 管理操作：创建、吊销、续期
//   - 安全事件：异地登录、异常行为
//
// 参数：
//   - licenseID: 关联的 License ID
//   - action: 操作类型
//   - detail: 操作详情
//   - ip: 操作来源 IP
//
// 返回：
//   - error: 记录失败错误
func (e *AdminEngine) AuditLog(licenseID, action, detail, ip string) error {
	return e.db.Create(&model.AuditLog{
		ID:        uuid.New().String(),
		LicenseID: licenseID,
		Action:    action,
		Detail:    detail,
		IP:        ip,
	}).Error
}

// GetAuditLogs 获取审计日志。
//
// 返回指定 License 的所有审计日志，按时间倒序排列。
//
// 参数：
//   - licenseID: License ID
//
// 返回：
//   - []model.AuditLog: 审计日志列表
//   - error: 查询失败错误
func (e *AdminEngine) GetAuditLogs(licenseID string) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	e.db.Where("license_id = ?", licenseID).Order("created_at DESC").Find(&logs)
	return logs, nil
}
