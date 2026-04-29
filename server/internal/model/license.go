// Package model 定义 Cerberus 的 GORM 数据模型。
//
// 该包包含以下核心模型：
//   - License: 软件授权许可证
//   - Machine: 绑定的机器
//   - AuditLog: 审计日志
package model

import (
	"time"

	"cerberus.dev/pkg/types"

	"gorm.io/gorm"
)

// License 软件授权许可证。
//
// License 是系统的核心实体，定义了授权的基本属性：
//   - 产品信息：名称、产品标识、发行者
//   - 绑定限制：最大机器数、IP 绑定开关
//   - 有效期：起始时间、截止时间
//   - 状态：活跃、吊销、过期、暂停
//   - 换绑控制：最大换绑次数、已用换绑次数
//   - 地理位置：首次激活时记录（Key 级别异地检测）
type License struct {
	ID              string             `gorm:"primaryKey;size:36" json:"id"`
	Name            string             `gorm:"size:255;not null" json:"name"`
	Product         string             `gorm:"size:255;not null" json:"product"`
	Issuer          string             `gorm:"size:255" json:"issuer"`
	MaxMachines     int                `gorm:"not null;default:1" json:"max_machines"`
	IPBindingEnabled bool              `gorm:"default:false" json:"ip_binding_enabled"`
	ValidFrom       int64              `gorm:"not null" json:"valid_from"`
	ValidUntil      int64              `gorm:"not null" json:"valid_until"`
	DurationSec     int64              `gorm:"not null" json:"duration_sec"`
	Status          types.LicenseStatus `gorm:"size:32;not null;default:active" json:"status"`
	Machines        []Machine          `gorm:"foreignKey:LicenseID" json:"machines,omitempty"`
	MaxUnbindCount  int                `gorm:"not null;default:3" json:"max_unbind_count"`
	UnbindCount     int                `gorm:"default:0" json:"unbind_count"`
	GeoCountry      string             `gorm:"size:8" json:"geo_country,omitempty"`
	GeoRegion       string             `gorm:"size:32" json:"geo_region,omitempty"`
	GeoCity         string             `gorm:"size:64" json:"geo_city,omitempty"`
	GeoLatitude     float64            `json:"geo_latitude,omitempty"`
	GeoLongitude    float64            `json:"geo_longitude,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	DeletedAt       gorm.DeletedAt     `gorm:"index" json:"-"`
}

// TableName 返回 License 表名。
func (License) TableName() string { return "licenses" }

// Machine 绑定的机器。
//
// Machine 存储每台绑定机器的详细信息：
//   - 身份标识：机器指纹、主机名
//   - 环境信息：操作系统、架构
//   - 网络信息：IP 地址、IP 绑定
//   - 状态信息：最后活跃时间、当前状态
type Machine struct {
	ID          string              `gorm:"primaryKey;size:36" json:"id"`
	LicenseID   string              `gorm:"size:36;index;not null" json:"license_id"`
	Fingerprint string              `gorm:"size:255;not null;index" json:"fingerprint"`
	Hostname    string              `gorm:"size:255" json:"hostname"`
	OS          string              `gorm:"size:64" json:"os"`
	Arch        string              `gorm:"size:64" json:"arch"`
	IP          string              `gorm:"size:64" json:"ip"`
	IPBinding   string              `gorm:"size:64" json:"ip_binding"`
	LastSeen    time.Time           `json:"last_seen"`
	Status      types.MachineStatus `gorm:"size:32;not null;default:active" json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	DeletedAt   gorm.DeletedAt      `gorm:"index" json:"-"`
}

// TableName 返回 Machine 表名。
func (Machine) TableName() string { return "machines" }

// AuditLog 审计日志。
//
// AuditLog 记录所有关键操作，用于审计和问题追踪：
//   - 激活、验证、心跳等客户端操作
//   - 创建、吊销、续期等管理操作
//   - 换绑、异地检测等安全事件
type AuditLog struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	LicenseID string    `gorm:"size:36;index" json:"license_id"`
	Action    string    `gorm:"size:64;not null" json:"action"`
	Detail    string    `gorm:"size:1024" json:"detail"`
	IP        string    `gorm:"size:64" json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 返回 AuditLog 表名。
func (AuditLog) TableName() string { return "audit_logs" }
