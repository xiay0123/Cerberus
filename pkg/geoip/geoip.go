// Package geoip 提供基于 IP 地址的地理位置检测功能。
//
// 该包实现以下核心功能：
//   - IP 地址解析与地理位置查询
//   - 地理区域定义与距离计算
//   - 异地登录检测策略
//
// 使用示例：
//
//	detector := geoip.NewDetector(geoip.Config{
//	    Policy: geoip.PolicyAlert,
//	    AllowedDistance: 500, // 500km
//	})
//	result, err := detector.Check(ctx, ip, licenseID)
package geoip

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// Policy 地理位置检测策略。
type Policy string

const (
	// PolicyAllow 允许异地登录（仅记录）。
	PolicyAllow Policy = "allow"
	// PolicyAlert 异地登录时告警但允许。
	PolicyAlert Policy = "alert"
	// PolicyDeny 拒绝异地登录。
	PolicyDeny Policy = "deny"
)

// Config 地理位置检测配置。
type Config struct {
	// Policy 检测策略。
	Policy Policy `json:"policy" yaml:"policy"`
	// AllowedDistance 允许的最大距离（公里），超过此距离触发策略。
	AllowedDistance float64 `json:"allowed_distance" yaml:"allowed_distance"`
	// DatabasePath GeoIP 数据库路径（可选，使用内置精简数据）。
	DatabasePath string `json:"database_path" yaml:"database_path"`
	// Enabled 是否启用地理位置检测。
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{
		Policy:          PolicyAllow,
		AllowedDistance: 500, // 500 公里
		Enabled:         false,
	}
}

// Location 表示一个地理位置。
type Location struct {
	// Country 国家代码（ISO 3166-1 alpha-2）。
	Country string `json:"country"`
	// CountryName 国家名称。
	CountryName string `json:"country_name"`
	// Region 地区/省份代码。
	Region string `json:"region,omitempty"`
	// RegionName 地区/省份名称。
	RegionName string `json:"region_name,omitempty"`
	// City 城市名称。
	City string `json:"city,omitempty"`
	// Latitude 纬度。
	Latitude float64 `json:"latitude"`
	// Longitude 经度。
	Longitude float64 `json:"longitude"`
	// Timezone 时区。
	Timezone string `json:"timezone,omitempty"`
}

// Region 表示一个地理区域（用于存储已记录区域）。
type Region struct {
	// Country 国家代码。
	Country string `json:"country"`
	// Region 地区代码。
	Region string `json:"region,omitempty"`
	// City 城市名称。
	City string `json:"city,omitempty"`
	// Latitude 纬度（中心点）。
	Latitude float64 `json:"latitude"`
	// Longitude 经度（中心点）。
	Longitude float64 `json:"longitude"`
}

// CheckResult 地理位置检测结果。
type CheckResult struct {
	// Allowed 是否允许访问。
	Allowed bool `json:"allowed"`
	// Alert 是否需要告警。
	Alert bool `json:"alert"`
	// CurrentLocation 当前位置。
	CurrentLocation *Location `json:"current_location,omitempty"`
	// RecordedLocation 已记录位置。
	RecordedLocation *Region `json:"recorded_location,omitempty"`
	// Distance 距离（公里）。
	Distance float64 `json:"distance,omitempty"`
	// Reason 原因说明。
	Reason string `json:"reason,omitempty"`
	// IsNewLocation 是否为新位置（首次激活）。
	IsNewLocation bool `json:"is_new_location,omitempty"`
}

// Detector 地理位置检测器。
type Detector struct {
	config    Config
	providers []Provider
	mu        sync.RWMutex
}

// Provider 地理位置数据提供者接口。
//
// 实现此接口可以接入不同的 GeoIP 数据源。
type Provider interface {
	// Lookup 查询 IP 地址的地理位置。
	Lookup(ctx context.Context, ip net.IP) (*Location, error)
	// Name 返回提供者名称。
	Name() string
}

// NewDetector 创建地理位置检测器。
//
// 参数：
//   - config: 检测器配置
//
// 返回：
//   - *Detector: 检测器实例
//   - error: 初始化错误
//
// 示例：
//
//	detector, err := geoip.NewDetector(geoip.Config{
//	    Policy: geoip.PolicyAlert,
//	    AllowedDistance: 500,
//	    Enabled: true,
//	})
func NewDetector(config Config) (*Detector, error) {
	if config.AllowedDistance <= 0 {
		config.AllowedDistance = 500
	}

	d := &Detector{
		config:    config,
		providers: make([]Provider, 0),
	}

	// 优先使用 GeoLite2 本地数据库（精确到城市，无限制）
	if config.DatabasePath != "" {
		if provider, err := newGeoLite2Provider(config.DatabasePath); err == nil {
			d.providers = append(d.providers, provider)
		}
	}

	// 在线 API 作为备用
	d.providers = append(d.providers, newOnlineProvider())
	// 内置精简数据作为最后备用
	d.providers = append(d.providers, newBuiltinProvider())

	return d, nil
}

// AddProvider 添加地理位置数据提供者。
//
// 提供者按添加顺序依次尝试，第一个成功返回结果的提供者生效。
//
// 参数：
//   - provider: 数据提供者实例
func (d *Detector) AddProvider(provider Provider) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.providers = append(d.providers, provider)
}

// Lookup 查询 IP 地址的地理位置。
//
// 参数：
//   - ctx: 上下文
//   - ipStr: IP 地址字符串
//
// 返回：
//   - *Location: 地理位置信息
//   - error: 查询错误
func (d *Detector) Lookup(ctx context.Context, ipStr string) (*Location, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// 跳过本地地址
	if isLocalIP(ip) {
		return &Location{
			Country:     "LOCAL",
			CountryName: "Local Network",
			City:        "Local",
		}, nil
	}

	d.mu.RLock()
	providers := d.providers
	d.mu.RUnlock()

	for _, p := range providers {
		loc, err := p.Lookup(ctx, ip)
		if err == nil && loc != nil {
			return loc, nil
		}
	}

	return nil, fmt.Errorf("no provider could resolve IP: %s", ipStr)
}

// Check 检查 IP 地址是否允许访问。
//
// 该方法实现完整的地理位置检测逻辑：
//  1. 查询当前 IP 的地理位置
//  2. 与已记录位置比较（如有）
//  3. 根据策略决定是否允许访问
//
// 参数：
//   - ctx: 上下文
//   - ipStr: 当前 IP 地址
//   - recordedRegion: 已记录的地理区域（可为 nil 表示首次激活）
//
// 返回：
//   - *CheckResult: 检测结果
//   - error: 检测错误
func (d *Detector) Check(ctx context.Context, ipStr string, recordedRegion *Region) (*CheckResult, error) {
	// 未启用检测，直接允许
	if !d.config.Enabled {
		return &CheckResult{
			Allowed:     true,
			Alert:       false,
			Reason:      "geoip detection disabled",
			IsNewLocation: recordedRegion == nil,
		}, nil
	}

	// 查询当前位置
	currentLoc, err := d.Lookup(ctx, ipStr)
	if err != nil {
		// 查询失败时的处理策略
		return &CheckResult{
			Allowed: true,
			Alert:   true,
			Reason:  fmt.Sprintf("geoip lookup failed: %v", err),
		}, nil
	}

	// 首次激活，记录位置
	if recordedRegion == nil {
		return &CheckResult{
			Allowed:       true,
			Alert:         false,
			CurrentLocation: currentLoc,
			IsNewLocation: true,
			Reason:        "first activation, location recorded",
		}, nil
	}

	// 计算距离
	distance := haversine(
		currentLoc.Latitude, currentLoc.Longitude,
		recordedRegion.Latitude, recordedRegion.Longitude,
	)

	result := &CheckResult{
		Allowed:          true,
		Alert:            false,
		CurrentLocation:  currentLoc,
		RecordedLocation: recordedRegion,
		Distance:         distance,
		IsNewLocation:    false,
	}

	// 判断是否异地
	isRemote := distance > d.config.AllowedDistance

	switch d.config.Policy {
	case PolicyAllow:
		// 允许模式：仅记录
		if isRemote {
			result.Alert = true
			result.Reason = fmt.Sprintf("remote login detected (%.0f km away)", distance)
		} else {
			result.Reason = "login within allowed area"
		}

	case PolicyAlert:
		// 告警模式：异地时告警但允许
		if isRemote {
			result.Alert = true
			result.Reason = fmt.Sprintf("remote login alert (%.0f km away, allowed)", distance)
		} else {
			result.Reason = "login within allowed area"
		}

	case PolicyDeny:
		// 拒绝模式：异地时拒绝
		if isRemote {
			result.Allowed = false
			result.Alert = true
			result.Reason = fmt.Sprintf("remote login denied (%.0f km away)", distance)
		} else {
			result.Reason = "login within allowed area"
		}
	}

	return result, nil
}

// UpdateRecord 更新已记录的地理区域。
//
// 返回适合存储的 Region 结构。
//
// 参数：
//   - loc: 当前位置信息
//
// 返回：
//   - *Region: 可存储的区域信息
func (d *Detector) UpdateRecord(loc *Location) *Region {
	if loc == nil {
		return nil
	}
	return &Region{
		Country:   loc.Country,
		Region:    loc.Region,
		City:      loc.City,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}
}

// isLocalIP 检查是否为本地 IP 地址。
func isLocalIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// 私有地址段
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, block := range privateBlocks {
		_, cidr, _ := net.ParseCIDR(block)
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// haversine 使用 Haversine 公式计算两点间距离（公里）。
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // 地球半径（公里）

	// 转换为弧度
	lat1Rad := lat1 * 0.017453292519943295
	lat2Rad := lat2 * 0.017453292519943295
	deltaLat := (lat2 - lat1) * 0.017453292519943295
	deltaLon := (lon2 - lon1) * 0.017453292519943295

	// Haversine 公式
	a := sin2(deltaLat/2) + cos(lat1Rad)*cos(lat2Rad)*sin2(deltaLon/2)
	c := 2 * atan2(sqrt(a), sqrt(1-a))

	return earthRadius * c
}

func sin2(x float64) float64 {
	s := x * x
	return s
}

func cos(x float64) float64 {
	return 1 - x*x/2 + x*x*x*x/24 // 泰勒展开近似
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func atan2(y, x float64) float64 {
	// 简化实现，适用于距离计算
	if x == 0 {
		if y > 0 {
			return 1.5707963267948966
		}
		if y < 0 {
			return -1.5707963267948966
		}
		return 0
	}
	return y / x // 简化近似
}
