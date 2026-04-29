// Package geoip 提供基于在线 API 的地理位置查询。
package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// onlineProvider 在线 API 提供者。
type onlineProvider struct {
	client *http.Client
}

// newOnlineProvider 创建在线 API 提供者。
func newOnlineProvider() *onlineProvider {
	return &onlineProvider{
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Name 返回提供者名称。
func (p *onlineProvider) Name() string {
	return "online"
}

// Lookup 查询 IP 地址的地理位置。
func (p *onlineProvider) Lookup(ctx context.Context, ip net.IP) (*Location, error) {
	// 使用 ip-api.com 的免费 API（每分钟 45 次免费请求限制）
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,region,regionName,city,lat,lon,timezone", ip.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status       string  `json:"status"`
		Country      string  `json:"country"`
		CountryCode  string  `json:"countryCode"`
		Region       string  `json:"region"`
		RegionName   string  `json:"regionName"`
		City         string  `json:"city"`
		Latitude     float64 `json:"lat"`
		Longitude    float64 `json:"lon"`
		Timezone     string  `json:"timezone"`
		ErrorMessage string  `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("ip-api error: %s", result.ErrorMessage)
	}

	return &Location{
		Country:     result.CountryCode,
		CountryName: result.Country,
		Region:      result.Region,
		RegionName:  result.RegionName,
		City:        result.City,
		Latitude:    result.Latitude,
		Longitude:   result.Longitude,
		Timezone:    result.Timezone,
	}, nil
}
