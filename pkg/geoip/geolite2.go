// Package geoip 提供 GeoLite2 数据库查询支持。
package geoip

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/oschwald/geoip2-golang"
)

// geoLite2Provider GeoLite2 数据库提供者。
type geoLite2Provider struct {
	db *geoip2.Reader
}

// newGeoLite2Provider 创建 GeoLite2 提供者。
//
// 参数：
//   - dbPath: GeoLite2-City.mmdb 数据库文件路径
//
// 返回：
//   - *geoLite2Provider: 提供者实例
//   - error: 初始化错误
func newGeoLite2Provider(dbPath string) (*geoLite2Provider, error) {
	// 检查文件是否存在
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("GeoLite2 database not found: %s", dbPath)
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoLite2 database: %w", err)
	}

	return &geoLite2Provider{db: db}, nil
}

// Name 返回提供者名称。
func (p *geoLite2Provider) Name() string {
	return "geolite2"
}

// Lookup 查询 IP 地址的地理位置。
//
// 使用 GeoLite2 数据库查询，精确到城市级别。
func (p *geoLite2Provider) Lookup(_ context.Context, ip net.IP) (*Location, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	record, err := p.db.City(ip)
	if err != nil {
		return nil, err
	}

	// 获取国家名称
	countryName := record.Country.Names["en"]
	if cn, ok := record.Country.Names["zh-CN"]; ok && cn != "" {
		countryName = cn
	}

	// 获取省份/地区名称
	regionName := ""
	regionCode := ""
	if len(record.Subdivisions) > 0 {
		regionCode = record.Subdivisions[0].IsoCode
		regionName = record.Subdivisions[0].Names["en"]
		if cn, ok := record.Subdivisions[0].Names["zh-CN"]; ok && cn != "" {
			regionName = cn
		}
	}

	// 获取城市名称
	cityName := record.City.Names["en"]
	if cn, ok := record.City.Names["zh-CN"]; ok && cn != "" {
		cityName = cn
	}

	return &Location{
		Country:     record.Country.IsoCode,
		CountryName: countryName,
		Region:      regionCode,
		RegionName:  regionName,
		City:        cityName,
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
		Timezone:    record.Location.TimeZone,
	}, nil
}

// Close 关闭数据库连接。
func (p *geoLite2Provider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
