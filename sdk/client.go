// Package sdk 提供 Cerberus Go SDK。
//
// 该 SDK 封装了 Cerberus Server 的 HTTP API，提供便捷的客户端调用方式。
// 支持以下功能：
//   - 机器激活（Activate）
//   - 在线验证（Verify）
//   - 心跳上报（Heartbeat）
//   - 自助换绑（Unbind）
//
// 使用示例：
//
//	client := sdk.NewClient("https://license.example.com")
//
//	// 激活
//	resp, err := client.Activate(ctx, "license-id", sdk.WithFingerprintAuto())
//
//	// 验证
//	result, err := client.Verify(ctx, "license-id", sdk.WithFingerprintAuto())
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cerberus.dev/pkg/fingerprint"
	"cerberus.dev/pkg/types"
)

// Client SDK 客户端。
//
// Client 封装了 HTTP 客户端，提供与 Cerberus Server 交互的方法。
type Client struct {
	// baseURL 服务端基础 URL。
	baseURL string
	// httpClient HTTP 客户端。
	httpClient *http.Client
}

// NewClient 创建 SDK 客户端。
//
// 参数：
//   - baseURL: Cerberus Server 基础 URL（如 https://license.example.com）
//
// 返回：
//   - *Client: SDK 客户端实例
//
// 示例：
//
//	client := sdk.NewClient("https://license.example.com")
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ActivateResponse 激活响应。
type ActivateResponse struct {
	// Machine 激活的机器信息。
	Machine *MachineInfo `json:"machine"`
	// Message 消息（可能包含告警信息）。
	Message string `json:"message,omitempty"`
	// GeoIPAlert 地理位置告警信息。
	GeoIPAlert string `json:"geoip_alert,omitempty"`
	// IsNewMachine 是否为新激活的机器。
	IsNewMachine bool `json:"is_new_machine"`
}

// MachineInfo 机器信息。
type MachineInfo struct {
	// ID 机器 ID。
	ID string `json:"id"`
	// LicenseID 所属 License ID。
	LicenseID string `json:"license_id"`
	// Fingerprint 机器指纹。
	Fingerprint string `json:"fingerprint"`
	// Hostname 主机名。
	Hostname string `json:"hostname"`
	// OS 操作系统。
	OS string `json:"os"`
	// Arch 系统架构。
	Arch string `json:"arch"`
	// IP IP 地址。
	IP string `json:"ip"`
	// Status 状态。
	Status string `json:"status"`
}

// Option SDK 选项函数。
type Option func(*options)

// options SDK 选项。
type options struct {
	// fingerprint 手动指定的指纹。
	fingerprint string
	// ip 手动指定的 IP。
	ip string
	// machineInfo 自动采集的机器信息。
	machineInfo *types.MachineInfo
}

// WithFingerprint 指定机器指纹。
//
// 使用场景：
//   - 测试时模拟不同机器
//   - 使用自定义指纹算法
//
// 参数：
//   - fp: 机器指纹字符串
//
// 返回：
//   - Option: 选项函数
func WithFingerprint(fp string) Option {
	return func(o *options) {
		o.fingerprint = fp
	}
}

// WithFingerprintAuto 自动采集机器指纹。
//
// 调用 fingerprint 包自动采集当前机器的硬件信息，
// 生成唯一的机器指纹。
//
// 返回：
//   - Option: 选项函数
func WithFingerprintAuto() Option {
	return func(o *options) {
		info, err := collectFingerprint()
		if err == nil {
			o.fingerprint = info.Fingerprint
			o.machineInfo = info
		}
	}
}

// WithIP 指定 IP 地址。
//
// 使用场景：
//   - 代理环境下指定真实 IP
//   - 测试时模拟不同 IP
//
// 参数：
//   - ip: IP 地址
//
// 返回：
//   - Option: 选项函数
func WithIP(ip string) Option {
	return func(o *options) {
		o.ip = ip
	}
}

// WithPublicIP 自动获取公网 IP。
//
// 通过访问外部服务获取当前机器的公网 IP 地址。
// 使用以下服务（按优先级尝试）：
//   - ifconfig.me
//   - icanhazip.com
//   - ipecho.net
//
// 返回：
//   - Option: 选项函数
func WithPublicIP() Option {
	return func(o *options) {
		if ip, err := getPublicIP(); err == nil {
			o.ip = ip
		}
	}
}

// getPublicIP 获取公网 IP 地址。
//
// 优先获取 IPv4，如果没有则获取 IPv6。
// 尝试多个服务确保可靠性。
func getPublicIP() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// 先尝试获取 IPv4（使用仅支持 IPv4 的服务）
	ipv4Services := []string{
		"https://api.ipify.org?format=text",
		"https://ipv4.icanhazip.com",
		"https://v4.ident.me",
	}

	for _, url := range ipv4Services {
		if ip, err := fetchIP(client, url); err == nil && isIPv4(ip) {
			return ip, nil
		}
	}

	// 如果没有 IPv4，尝试通用服务（可能返回 IPv4 或 IPv6）
	generalServices := []string{
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://ipecho.net/plain",
	}

	for _, url := range generalServices {
		if ip, err := fetchIP(client, url); err == nil {
			// 优先返回 IPv4
			if isIPv4(ip) {
				return ip, nil
			}
			// 记录 IPv6 作为备选
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get public IP from all services")
}

// fetchIP 从指定 URL 获取 IP
func fetchIP(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(data))
	if ip == "" {
		return "", fmt.Errorf("empty response")
	}
	return ip, nil
}

// isIPv4 检查是否为 IPv4 地址
func isIPv4(ip string) bool {
	// 简单检查：IPv4 包含点，不包含冒号
	return strings.Contains(ip, ".") && !strings.Contains(ip, ":")
}

// applyOptions 应用选项。
func applyOptions(opts ...Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Activate 激活机器。
//
// 激活流程：
//  1. 收集机器信息（指纹、主机名、操作系统等）
//  2. 向服务端发送激活请求
//  3. 服务端验证 License 并绑定机器
//  4. 返回激活结果（可能包含地理位置告警）
//
// 参数：
//   - ctx: 上下文
//   - licenseID: License ID
//   - opts: 可选参数（指纹、IP 等）
//
// 返回：
//   - *ActivateResponse: 激活响应
//   - error: 激活失败错误
//
// 示例：
//
//	resp, err := client.Activate(ctx, "license-id", sdk.WithFingerprintAuto())
func (c *Client) Activate(ctx context.Context, licenseID string, opts ...Option) (*ActivateResponse, error) {
	options := applyOptions(opts...)

	reqBody := map[string]interface{}{
		"license_id": licenseID,
	}
	if options.fingerprint != "" {
		reqBody["fingerprint"] = options.fingerprint
	}
	if options.machineInfo != nil {
		reqBody["hostname"] = options.machineInfo.Hostname
		reqBody["os"] = options.machineInfo.OS
		reqBody["arch"] = options.machineInfo.Arch
	}
	// IP 优先级：WithIP/WithPublicIP > machineInfo.IP
	if options.ip != "" {
		reqBody["ip"] = options.ip
	} else if options.machineInfo != nil && options.machineInfo.IP != "" {
		reqBody["ip"] = options.machineInfo.IP
	}

	var resp ActivateResponse
	if err := c.doPost(ctx, "/api/v1/activate", reqBody, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Verify 在线验证。
//
// 验证流程：
//  1. 向服务端发送验证请求
//  2. 服务端检查 License 状态、有效期
//  3. 如提供指纹，验证机器绑定状态
//  4. 返回验证结果
//
// 参数：
//   - ctx: 上下文
//   - licenseID: License ID
//   - opts: 可选参数（指纹、IP 等）
//
// 返回：
//   - *types.VerifyResult: 验证结果
//   - error: 验证失败错误
//
// 示例：
//
//	result, err := client.Verify(ctx, "license-id", sdk.WithFingerprintAuto())
//	if result.Valid {
//	    // License 有效
//	}
func (c *Client) Verify(ctx context.Context, licenseID string, opts ...Option) (*types.VerifyResult, error) {
	options := applyOptions(opts...)

	reqBody := map[string]interface{}{
		"license_id": licenseID,
	}
	if options.fingerprint != "" {
		reqBody["fingerprint"] = options.fingerprint
	}
	if options.ip != "" {
		reqBody["ip"] = options.ip
	}

	var result types.VerifyResult
	if err := c.doPost(ctx, "/api/v1/verify", reqBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Heartbeat 心跳上报。
//
// 心跳用于更新机器的活跃状态：
//   - 更新服务端的 last_seen 时间戳
//   - 更新当前 IP 地址
//
// 建议在应用程序运行期间定期发送心跳（如每 5 分钟）。
//
// 参数：
//   - ctx: 上下文
//   - licenseID: License ID
//   - opts: 可选参数（指纹、IP 等）
//
// 返回：
//   - error: 心跳失败错误
//
// 示例：
//
//	err := client.Heartbeat(ctx, "license-id", sdk.WithFingerprintAuto())
func (c *Client) Heartbeat(ctx context.Context, licenseID string, opts ...Option) error {
	options := applyOptions(opts...)

	reqBody := map[string]interface{}{
		"license_id":   licenseID,
		"fingerprint": options.fingerprint,
	}
	if options.ip != "" {
		reqBody["ip"] = options.ip
	}

	return c.doPost(ctx, "/api/v1/heartbeat", reqBody, nil)
}

// Unbind 换绑机器。
//
// 换绑允许用户自助解绑旧机器，以便绑定新机器。
// 注意：
//   - 每个 License 有换绑次数限制
//   - 超限后 License 会被暂停
//   - 需要提供旧机器的指纹
//
// 参数：
//   - ctx: 上下文
//   - licenseID: License ID
//   - oldFingerprint: 要解绑的机器指纹
//
// 返回：
//   - *types.UnbindMachineResult: 换绑结果
//   - error: 换绑失败错误
//
// 示例：
//
//	result, err := client.Unbind(ctx, "license-id", "old-fingerprint")
//	fmt.Printf("剩余换绑次数: %d\n", result.Remaining)
func (c *Client) Unbind(ctx context.Context, licenseID, oldFingerprint string) (*types.UnbindMachineResult, error) {
	reqBody := map[string]interface{}{
		"license_id":       licenseID,
		"old_fingerprint": oldFingerprint,
	}

	var result types.UnbindMachineResult
	if err := c.doPost(ctx, "/api/v1/unbind", reqBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// doPost 执行 POST 请求。
//
// 内部方法，封装了 HTTP POST 请求的通用逻辑：
//   - 构建请求 URL
//   - 序列化请求体
//   - 设置请求头
//   - 发送请求并处理响应
//   - 解析 API 响应格式
//
// 参数：
//   - ctx: 上下文
//   - path: API 路径
//   - reqBody: 请求体（可为 nil）
//   - respBody: 响应体指针（可为 nil）
//
// 返回：
//   - error: 请求失败错误
func (c *Client) doPost(ctx context.Context, path string, reqBody interface{}, respBody interface{}) error {
	url := c.baseURL + path

	var body io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var apiResp struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respData, &apiResp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if apiResp.Code != 0 {
		return fmt.Errorf("API error: %s (code=%d)", apiResp.Message, apiResp.Code)
	}

	if respBody != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, respBody); err != nil {
			return fmt.Errorf("parse data: %w", err)
		}
	}

	return nil
}

// collectFingerprint 采集机器指纹。
//
// 内部方法，调用 fingerprint 包采集当前机器的硬件信息。
func collectFingerprint() (*types.MachineInfo, error) {
	info, err := fingerprint.Collect()
	if err != nil {
		return nil, fmt.Errorf("collect fingerprint: %w", err)
	}
	return info, nil
}
