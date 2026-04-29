// Package geoip 提供内置的精简 GeoIP 数据。
//
// 该文件包含主要国家/地区的 IP 地址段映射，
// 用于在没有外部 GeoIP 数据库时提供基础的地理位置查询功能。
package geoip

import (
	"context"
	"net"
	"sync"
)

// builtinProvider 内置数据提供者。
type builtinProvider struct {
	countries map[string]*Location
	mu        sync.RWMutex
}

// newBuiltinProvider 创建内置数据提供者。
func newBuiltinProvider() *builtinProvider {
	return &builtinProvider{
		countries: getBuiltinData(),
	}
}

// Name 返回提供者名称。
func (p *builtinProvider) Name() string {
	return "builtin"
}

// Lookup 查询 IP 地址的地理位置。
//
// 内置提供者使用 IP 地址段前缀匹配，
// 精度有限，仅能定位到国家/地区级别。
//
// 参数：
//   - ctx: 上下文（当前未使用）
//   - ip: IP 地址
//
// 返回：
//   - *Location: 地理位置信息
//   - error: 查询错误
func (p *builtinProvider) Lookup(_ context.Context, ip net.IP) (*Location, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 提取 IP 前缀进行匹配
	ip4 := ip.To4()
	if ip4 == nil {
		// IPv6 暂不支持
		return nil, ErrNotFound
	}

	// 计算第一个字节用于快速定位
	firstByte := int(ip4[0])

	// 根据第一个字节范围确定国家
	// 这是一个简化实现，实际应用应使用完整数据库
	country := lookupCountryByFirstByte(firstByte)
	if loc, ok := p.countries[country]; ok {
		return loc, nil
	}

	return nil, ErrNotFound
}

// ErrNotFound 表示未找到地理位置数据。
var ErrNotFound = &GeoIPError{Message: "location not found"}

// GeoIPError GeoIP 错误类型。
type GeoIPError struct {
	Message string
}

// Error 实现 error 接口。
func (e *GeoIPError) Error() string {
	return e.Message
}

// lookupCountryByFirstByte 根据 IP 第一个字节估算国家。
//
// 这是一个高度简化的实现，仅用于演示目的。
// 实际生产环境应使用 MaxMind GeoIP2 或 IP2Location 等专业数据库。
func lookupCountryByFirstByte(b int) string {
	// 简化的 IP 分配映射
	// 注意：这只是一个粗略的估算，不准确！
	switch {
	case b >= 1 && b <= 9:
		return "US" // 美国
	case b >= 11 && b <= 22:
		return "US" // 美国
	case b >= 24 && b <= 41:
		return "US" // 美国
	case b >= 43 && b <= 63:
		return "US" // 美国
	case b >= 64 && b <= 71:
		return "US" // 美国
	case b >= 72 && b <= 77:
		return "US" // 美国
	case b >= 78 && b <= 79:
		return "US" // 美国
	case b >= 80 && b <= 95:
		return "GB" // 英国/欧洲
	case b >= 96 && b <= 111:
		return "US" // 美国
	case b >= 112 && b <= 126:
		return "US" // 美国
	case b >= 128 && b <= 171:
		return "US" // 美国
	case b >= 172 && b <= 175:
		return "US" // 美国（部分私有地址）
	case b >= 176 && b <= 191:
		return "US" // 美国
	case b >= 192 && b <= 195:
		return "DE" // 德国/欧洲
	case b >= 196 && b <= 199:
		return "US" // 美国
	case b >= 200 && b <= 201:
		return "BR" // 巴西
	case b >= 202 && b <= 203:
		return "CN" // 中国/亚太
	case b >= 210 && b <= 211:
		return "CN" // 中国
	case b >= 218 && b <= 223:
		return "CN" // 中国/亚太
	default:
		return "US" // 默认美国
	}
}

// getBuiltinData 获取内置国家位置数据。
//
// 返回各国中心点坐标，用于距离计算。
func getBuiltinData() map[string]*Location {
	return map[string]*Location{
		"CN": {
			Country:     "CN",
			CountryName: "China",
			Region:      "Beijing",
			RegionName:  "北京",
			City:        "Beijing",
			Latitude:    39.9042,
			Longitude:   116.4074,
			Timezone:    "Asia/Shanghai",
		},
		"US": {
			Country:     "US",
			CountryName: "United States",
			Region:      "Kansas",
			RegionName:  "Kansas",
			City:        "Lebanon",
			Latitude:    39.8283,
			Longitude:   -98.5795,
			Timezone:    "America/Chicago",
		},
		"GB": {
			Country:     "GB",
			CountryName: "United Kingdom",
			Region:      "England",
			RegionName:  "England",
			City:        "London",
			Latitude:    51.5074,
			Longitude:   -0.1278,
			Timezone:    "Europe/London",
		},
		"DE": {
			Country:     "DE",
			CountryName: "Germany",
			Region:      "Berlin",
			RegionName:  "Berlin",
			City:        "Berlin",
			Latitude:    52.5200,
			Longitude:   13.4050,
			Timezone:    "Europe/Berlin",
		},
		"JP": {
			Country:     "JP",
			CountryName: "Japan",
			Region:      "Tokyo",
			RegionName:  "Tokyo",
			City:        "Tokyo",
			Latitude:    35.6762,
			Longitude:   139.6503,
			Timezone:    "Asia/Tokyo",
		},
		"KR": {
			Country:     "KR",
			CountryName: "South Korea",
			Region:      "Seoul",
			RegionName:  "Seoul",
			City:        "Seoul",
			Latitude:    37.5665,
			Longitude:   126.9780,
			Timezone:    "Asia/Seoul",
		},
		"SG": {
			Country:     "SG",
			CountryName: "Singapore",
			Region:      "Singapore",
			RegionName:  "Singapore",
			City:        "Singapore",
			Latitude:    1.3521,
			Longitude:   103.8198,
			Timezone:    "Asia/Singapore",
		},
		"HK": {
			Country:     "HK",
			CountryName: "Hong Kong",
			Region:      "Hong Kong",
			RegionName:  "Hong Kong",
			City:        "Hong Kong",
			Latitude:    22.3193,
			Longitude:   114.1694,
			Timezone:    "Asia/Hong_Kong",
		},
		"TW": {
			Country:     "TW",
			CountryName: "Taiwan",
			Region:      "Taipei",
			RegionName:  "Taipei",
			City:        "Taipei",
			Latitude:    25.0330,
			Longitude:   121.5654,
			Timezone:    "Asia/Taipei",
		},
		"AU": {
			Country:     "AU",
			CountryName: "Australia",
			Region:      "Canberra",
			RegionName:  "Canberra",
			City:        "Canberra",
			Latitude:    -35.2809,
			Longitude:   149.1300,
			Timezone:    "Australia/Sydney",
		},
		"BR": {
			Country:     "BR",
			CountryName: "Brazil",
			Region:      "Brasilia",
			RegionName:  "Brasilia",
			City:        "Brasilia",
			Latitude:    -15.8267,
			Longitude:   -47.9218,
			Timezone:    "America/Sao_Paulo",
		},
		"IN": {
			Country:     "IN",
			CountryName: "India",
			Region:      "New Delhi",
			RegionName:  "New Delhi",
			City:        "New Delhi",
			Latitude:    28.6139,
			Longitude:   77.2090,
			Timezone:    "Asia/Kolkata",
		},
		"RU": {
			Country:     "RU",
			CountryName: "Russia",
			Region:      "Moscow",
			RegionName:  "Moscow",
			City:        "Moscow",
			Latitude:    55.7558,
			Longitude:   37.6173,
			Timezone:    "Europe/Moscow",
		},
		"FR": {
			Country:     "FR",
			CountryName: "France",
			Region:      "Paris",
			RegionName:  "Paris",
			City:        "Paris",
			Latitude:    48.8566,
			Longitude:   2.3522,
			Timezone:    "Europe/Paris",
		},
		"CA": {
			Country:     "CA",
			CountryName: "Canada",
			Region:      "Ottawa",
			RegionName:  "Ottawa",
			City:        "Ottawa",
			Latitude:    45.4215,
			Longitude:   -75.6972,
			Timezone:    "America/Toronto",
		},
	}
}
