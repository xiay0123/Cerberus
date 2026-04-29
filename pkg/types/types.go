// Package types 提供 Cerberus 共享的类型定义。
//
// 该包定义了 SDK 和 Server 共用的核心类型：
//   - LicenseStatus: License 状态枚举
//   - MachineStatus: 机器状态枚举
//   - VerifyResult: 验证结果结构
//   - UnbindMachineResult: 换绑结果结构
//   - MachineInfo: 机器信息结构
package types

// LicenseStatus 授权状态。
//
// License 可能处于以下状态：
//   - active: 活跃，正常使用
//   - revoked: 已吊销，无法恢复
//   - expired: 已过期，可续期
//   - suspended: 已暂停，可解封
type LicenseStatus string

const (
	// LicenseActive 活跃状态。
	LicenseActive LicenseStatus = "active"
	// LicenseRevoked 已吊销状态。
	LicenseRevoked LicenseStatus = "revoked"
	// LicenseExpired 已过期状态。
	LicenseExpired LicenseStatus = "expired"
	// LicenseSuspended 已暂停状态。
	LicenseSuspended LicenseStatus = "suspended"
)

// MachineStatus 机器状态。
//
// Machine 可能处于以下状态：
//   - active: 活跃，正常使用
//   - revoked: 已吊销
//   - stale: 长时间无心跳
type MachineStatus string

const (
	// MachineActive 活跃状态。
	MachineActive MachineStatus = "active"
	// MachineRevoked 已吊销状态。
	MachineRevoked MachineStatus = "revoked"
	// MachineStale 僵尸状态（长时间无心跳）。
	MachineStale MachineStatus = "stale"
)

// VerifyResult 验证结果。
//
// 包含 License 验证的完整结果信息：
//   - 验证状态（valid）
//   - License 基本信息（id、product）
//   - 有效期信息（expires_in、expires_at）
//   - 机器绑定信息（machine_id、max_machines）
//   - 失败原因（reason）
type VerifyResult struct {
	// Valid 是否有效。
	Valid bool `json:"valid"`
	// LicenseID License ID。
	LicenseID string `json:"license_id,omitempty"`
	// Product 产品标识。
	Product string `json:"product,omitempty"`
	// Reason 无效原因。
	Reason string `json:"reason,omitempty"`
	// ExpiresIn 剩余有效时间（秒）。
	ExpiresIn int64 `json:"expires_in,omitempty"`
	// ExpiresAt 过期时间戳。
	ExpiresAt int64 `json:"expires_at,omitempty"`
	// MachineID 机器 ID（已激活时）。
	MachineID string `json:"machine_id,omitempty"`
	// MaxMachines 最大机器数。
	MaxMachines int `json:"max_machines,omitempty"`
}

// UnbindMachineResult 换绑结果。
//
// 包含换绑操作的详细结果：
//   - 操作状态（success）
//   - 被吊销的机器 ID
//   - 剩余换绑次数
//   - 结果说明消息
type UnbindMachineResult struct {
	// Success 是否成功。
	Success bool `json:"success"`
	// MachineRevoked 被吊销的机器 ID。
	MachineRevoked string `json:"machine_revoked,omitempty"`
	// Remaining 剩余换绑次数。
	Remaining int `json:"remaining"`
	// Message 结果说明。
	Message string `json:"message"`
}

// MachineInfo 机器信息（指纹采集结果）。
//
// 包含机器的详细硬件和环境信息：
//   - Fingerprint: 机器指纹（硬件信息哈希）
//   - Hostname: 主机名
//   - OS: 操作系统
//   - Arch: 系统架构
//   - IP: IP 地址（可选）
type MachineInfo struct {
	// Fingerprint 机器指纹（硬件信息哈希）。
	Fingerprint string `json:"fingerprint"`
	// Hostname 主机名。
	Hostname string `json:"hostname"`
	// OS 操作系统（如 windows、linux、darwin）。
	OS string `json:"os"`
	// Arch 系统架构（如 amd64、arm64）。
	Arch string `json:"arch"`
	// IP IP 地址（可选）。
	IP string `json:"ip,omitempty"`
}
