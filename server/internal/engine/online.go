// Package engine 提供 Cerberus 在线验证引擎。
//
// 该包实现 License 的在线验证核心逻辑：
//   - 机器激活与绑定
//   - 实时在线验证
//   - 心跳上报与状态监控
//   - 客户端自助换绑
//   - 地理位置检测（Key 级别异地登录告警）
package engine

import (
	"fmt"
	"time"

	"cerberus.dev/pkg/geoip"
	"cerberus.dev/pkg/types"
	"cerberus.dev/server/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OnlineEngine 在线验证引擎。
//
// 负责 License 的在线验证全流程，包括：
//   - 激活：绑定机器到 License
//   - 验证：检查 License 有效性
//   - 心跳：更新机器活跃状态
//   - 换绑：解绑旧机器以激活新机器
//   - GeoIP：检测异地登录
type OnlineEngine struct {
	db            *gorm.DB           // 数据库连接
	maxMachines   int                // 默认最大机器数
	heartbeatTTL  time.Duration      // 心跳超时时间（暂未使用）
	geoipDetector *geoip.Detector    // 地理位置检测器
}

// NewOnlineEngine 创建在线验证引擎实例。
//
// 参数：
//   - db: GORM 数据库连接
//   - maxMachines: 默认最大绑定机器数
//   - heartbeatTTL: 心跳超时时间
//
// 返回：
//   - *OnlineEngine: 引擎实例
func NewOnlineEngine(db *gorm.DB, maxMachines int, heartbeatTTL time.Duration) *OnlineEngine {
	return &OnlineEngine{
		db:           db,
		maxMachines:  maxMachines,
		heartbeatTTL: heartbeatTTL,
	}
}

// SetGeoIPDetector 设置地理位置检测器。
//
// 参数：
//   - detector: GeoIP 检测器实例
func (e *OnlineEngine) SetGeoIPDetector(detector *geoip.Detector) {
	e.geoipDetector = detector
}

// ActivateParams 激活参数。
//
// 包含激活机器所需的全部信息。
type ActivateParams struct {
	Fingerprint string // 机器指纹（必填，硬件信息哈希）
	Hostname    string // 主机名
	OS          string // 操作系统
	Arch        string // 系统架构
	IP          string // IP 地址
}

// ActivateResult 激活结果。
//
// 返回激活操作的详细结果信息。
type ActivateResult struct {
	Machine      *model.Machine // 激活的机器信息
	GeoIPAlert   string         // 地理位置告警信息（异地登录）
	IsNewMachine bool           // 是否为新激活的机器
}

// Activate 激活机器。
//
// 激活流程：
//  1. 验证 License 存在且状态为活跃
//  2. 检查 License 是否在有效期内
//  3. 已存在的机器：更新信息
//  4. 新机器：检查数量限制后创建
//  5. GeoIP：首次激活记录位置，后续检测距离
//
// 参数：
//   - licenseID: License ID
//   - params: 激活参数
//
// 返回：
//   - *ActivateResult: 激活结果
//   - error: 激活失败错误
func (e *OnlineEngine) Activate(licenseID string, params ActivateParams) (*ActivateResult, error) {
	var l model.License
	if err := e.db.Where("id = ? AND status = ?", licenseID, types.LicenseActive).First(&l).Error; err != nil {
		return nil, fmt.Errorf("license not found or not active")
	}

	now := time.Now().Unix()
	if now < l.ValidFrom {
		return nil, fmt.Errorf("license not yet valid")
	}
	if now > l.ValidUntil {
		return nil, fmt.Errorf("license expired")
	}

	var existing model.Machine
	err := e.db.Where("license_id = ? AND fingerprint = ?", licenseID, params.Fingerprint).First(&existing).Error

	result := &ActivateResult{}

	if err == nil {
		// 已存在的机器
		if existing.Status == types.MachineRevoked {
			if l.UnbindCount >= l.MaxUnbindCount {
				return nil, fmt.Errorf("unbind count exceeded (%d/%d), cannot reactivate this device", l.UnbindCount, l.MaxUnbindCount)
			}
			existing.Status = types.MachineActive
			existing.IPBinding = ""
			if l.IPBindingEnabled && params.IP != "" {
				existing.IPBinding = params.IP
			}
		}
		existing.LastSeen = time.Now()
		existing.IP = params.IP
		existing.Hostname = params.Hostname
		existing.OS = params.OS
		existing.Arch = params.Arch

		// 地理位置检测
		if e.geoipDetector != nil && params.IP != "" && l.GeoCountry != "" {
			result.GeoIPAlert, _ = e.checkGeoIP(params.IP, &l)
		}

		e.db.Save(&existing)
		result.Machine = &existing
		return result, nil
	}

	// 新机器激活
	var activeCount int64
	e.db.Model(&model.Machine{}).
		Where("license_id = ? AND status = ?", licenseID, types.MachineActive).
		Count(&activeCount)

	if activeCount >= int64(l.MaxMachines) {
		return nil, fmt.Errorf("max machines (%d) reached, please unbind an existing device", l.MaxMachines)
	}

	ipBinding := ""
	if l.IPBindingEnabled && params.IP != "" {
		ipBinding = params.IP
	}

	m := &model.Machine{
		ID:          uuid.New().String(),
		LicenseID:   licenseID,
		Fingerprint: params.Fingerprint,
		Hostname:    params.Hostname,
		OS:          params.OS,
		Arch:        params.Arch,
		IP:          params.IP,
		IPBinding:   ipBinding,
		LastSeen:    time.Now(),
		Status:      types.MachineActive,
	}

	// 地理位置检测：首次激活记录位置，后续检测距离
	if e.geoipDetector != nil && params.IP != "" {
		if l.GeoCountry == "" {
			e.recordGeoIP(params.IP, &l)
		} else {
			result.GeoIPAlert, _ = e.checkGeoIP(params.IP, &l)
		}
	}

	if err := e.db.Create(m).Error; err != nil {
		return nil, fmt.Errorf("save machine: %w", err)
	}

	result.Machine = m
	result.IsNewMachine = true
	return result, nil
}

// Verify 在线验证。
//
// 验证流程：
//  1. 查询 License 信息
//  2. 检查 License 状态（吊销、过期等）
//  3. 验证机器绑定状态（如提供指纹）
//  4. 验证 IP 地址（如启用 IP 绑定）
//
// 参数：
//   - licenseID: License ID
//   - fingerprint: 机器指纹（可选）
//   - ip: 当前 IP 地址（可选）
//
// 返回：
//   - types.VerifyResult: 验证结果
func (e *OnlineEngine) Verify(licenseID string, fingerprint string, ip string) types.VerifyResult {
	var l model.License
	if err := e.db.Where("id = ?", licenseID).First(&l).Error; err != nil {
		return types.VerifyResult{Valid: false, Reason: "license not found"}
	}

	if l.Status == types.LicenseRevoked {
		return types.VerifyResult{Valid: false, LicenseID: l.ID, Product: l.Product, Reason: "license revoked", MaxMachines: l.MaxMachines}
	}

	now := time.Now().Unix()
	if now < l.ValidFrom {
		return types.VerifyResult{Valid: false, LicenseID: l.ID, Product: l.Product, Reason: "license not yet valid", MaxMachines: l.MaxMachines}
	}
	if now > l.ValidUntil {
		e.db.Model(&l).Update("status", types.LicenseExpired)
		return types.VerifyResult{Valid: false, LicenseID: l.ID, Product: l.Product, Reason: "license expired", ExpiresAt: l.ValidUntil, MaxMachines: l.MaxMachines}
	}

	var m model.Machine
	if fingerprint != "" {
		err := e.db.Where("license_id = ? AND fingerprint = ? AND status = ?", licenseID, fingerprint, types.MachineActive).First(&m).Error
		if err != nil {
			return types.VerifyResult{Valid: false, LicenseID: l.ID, Product: l.Product, Reason: "machine not activated or revoked", MaxMachines: l.MaxMachines}
		}

		if m.IPBinding != "" && m.IPBinding != ip {
			return types.VerifyResult{Valid: false, LicenseID: l.ID, Product: l.Product, Reason: "IP mismatch", MaxMachines: l.MaxMachines}
		}
	}

	return types.VerifyResult{Valid: true, LicenseID: l.ID, Product: l.Product, ExpiresIn: l.ValidUntil - now, ExpiresAt: l.ValidUntil, MachineID: m.ID, MaxMachines: l.MaxMachines}
}

// Heartbeat 心跳上报。
//
// 更新机器的活跃状态并检测异地登录。
//
// 参数：
//   - licenseID: License ID
//   - fingerprint: 机器指纹
//   - ip: 当前 IP 地址
//
// 返回：
//   - error: 心跳失败错误
func (e *OnlineEngine) Heartbeat(licenseID, fingerprint, ip string) error {
	var m model.Machine
	err := e.db.Where("license_id = ? AND fingerprint = ? AND status = ?", licenseID, fingerprint, types.MachineActive).First(&m).Error
	if err != nil {
		return fmt.Errorf("machine not found or revoked")
	}

	var l model.License
	if err := e.db.Where("id = ? AND status = ?", licenseID, types.LicenseActive).First(&l).Error; err != nil {
		return fmt.Errorf("license not found or not active")
	}

	if time.Now().Unix() > l.ValidUntil {
		return fmt.Errorf("license expired")
	}

	// 地理位置检测
	if e.geoipDetector != nil && ip != "" && l.GeoCountry != "" {
		if alert, _ := e.checkGeoIP(ip, &l); alert != "" {
			e.db.Create(&model.AuditLog{
				ID:        uuid.New().String(),
				LicenseID: licenseID,
				Action:    "geoip_alert",
				Detail:    alert,
				IP:        ip,
			})
		}
	}

	return e.db.Model(&m).Updates(map[string]interface{}{"last_seen": time.Now(), "ip": ip}).Error
}

// UnbindMachine 客户端自助换绑。
//
// 换绑流程：
//  1. 验证 License 存在且未被吊销
//  2. 检查换绑次数限制
//  3. 将指定机器标记为已吊销
//  4. 递增换绑计数
//
// 参数：
//   - licenseID: License ID
//   - oldFingerprint: 要解绑的机器指纹
//
// 返回：
//   - *types.UnbindMachineResult: 换绑结果
//   - error: 换绑失败错误
func (e *OnlineEngine) UnbindMachine(licenseID, oldFingerprint string) (*types.UnbindMachineResult, error) {
	var l model.License
	if err := e.db.Where("id = ?", licenseID).First(&l).Error; err != nil {
		return nil, fmt.Errorf("license not found")
	}

	if l.Status == types.LicenseRevoked {
		return nil, fmt.Errorf("license is revoked")
	}
	if l.Status == types.LicenseSuspended {
		return nil, fmt.Errorf("license is suspended")
	}

	if l.UnbindCount >= l.MaxUnbindCount {
		e.db.Model(&l).Update("status", types.LicenseSuspended)
		return nil, fmt.Errorf("unbind limit exceeded (%d/%d), license suspended", l.UnbindCount, l.MaxUnbindCount)
	}

	var m model.Machine
	if err := e.db.Where("license_id = ? AND fingerprint = ? AND status = ?", licenseID, oldFingerprint, types.MachineActive).First(&m).Error; err != nil {
		return nil, fmt.Errorf("machine not found or already revoked")
	}

	e.db.Model(&m).Update("status", types.MachineRevoked)

	newCount := l.UnbindCount + 1
	e.db.Model(&l).Update("unbind_count", newCount)

	return &types.UnbindMachineResult{
		Success:        true,
		MachineRevoked: m.ID,
		Remaining:      l.MaxUnbindCount - newCount,
		Message:        fmt.Sprintf("machine unbound, %d unbind operations remaining", l.MaxUnbindCount-newCount),
	}, nil
}

// checkGeoIP 检查地理位置并返回告警信息。
//
// 比较当前 IP 与 License 注册位置的距离，
// 超过配置阈值时返回告警信息。
//
// 参数：
//   - ip: 当前 IP 地址
//   - l: License 实例
//
// 返回：
//   - string: 告警信息（无告警时为空）
//   - error: 检测错误
func (e *OnlineEngine) checkGeoIP(ip string, l *model.License) (string, error) {
	if e.geoipDetector == nil || l.GeoCountry == "" {
		return "", nil
	}

	result, err := e.geoipDetector.Check(nil, ip, &geoip.Region{
		Country:   l.GeoCountry,
		Region:    l.GeoRegion,
		City:      l.GeoCity,
		Latitude:  l.GeoLatitude,
		Longitude: l.GeoLongitude,
	})
	if err != nil {
		return "", err
	}

	if result.Alert {
		return result.Reason, nil
	}
	return "", nil
}

// recordGeoIP 记录 License 的地理位置。
//
// 首次激活时查询 IP 的地理位置并保存到 License。
//
// 参数：
//   - ip: IP 地址
//   - l: License 实例
func (e *OnlineEngine) recordGeoIP(ip string, l *model.License) {
	if e.geoipDetector == nil {
		return
	}

	loc, err := e.geoipDetector.Lookup(nil, ip)
	if err != nil {
		return
	}

	l.GeoCountry = loc.Country
	l.GeoRegion = loc.Region
	l.GeoCity = loc.City
	l.GeoLatitude = loc.Latitude
	l.GeoLongitude = loc.Longitude

	e.db.Model(l).Updates(map[string]interface{}{
		"geo_country":   l.GeoCountry,
		"geo_region":    l.GeoRegion,
		"geo_city":      l.GeoCity,
		"geo_latitude":  l.GeoLatitude,
		"geo_longitude": l.GeoLongitude,
	})
}
