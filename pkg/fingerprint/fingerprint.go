// Package fingerprint 提供跨平台的机器指纹采集功能。
//
// 机器指纹用于唯一标识一台物理机器，基于以下硬件信息：
//   - CPU ID / ProcessorId
//   - 磁盘序列号
//   - 主板 UUID
//   - 网卡 MAC 地址
//
// 支持的平台：
//   - Windows（使用 WMIC 命令）
//   - Linux（读取 /proc 和 /sys 文件系统）
//   - macOS（使用 ioreg 和 sysctl 命令）
//
// 使用示例：
//
//	info, err := fingerprint.Collect()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Fingerprint: %s\n", info.Fingerprint)
//	fmt.Printf("Hostname: %s\n", info.Hostname)
//	fmt.Printf("OS: %s\n", info.OS)
//	fmt.Printf("Arch: %s\n", info.Arch)
package fingerprint

import (
	"crypto/sha256"
	"fmt"
	"runtime"

	"cerberus.dev/pkg/types"
)

// Collect 采集当前机器的指纹信息。
//
// 采集流程：
//  1. 获取操作系统和架构信息
//  2. 获取主机名
//  3. 根据平台调用相应的硬件信息采集函数
//  4. 将硬件 ID 拼接并计算 SHA256 哈希作为指纹
//
// 返回：
//   - *types.MachineInfo: 机器信息，包含指纹、主机名、操作系统、架构
//   - error: 采集失败时返回错误
//
// 示例：
//
//	info, err := fingerprint.Collect()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// 使用指纹激活或验证
//	client.Activate(ctx, licenseID, sdk.WithFingerprint(info.Fingerprint))
func Collect() (*types.MachineInfo, error) {
	info := &types.MachineInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// 获取主机名
	hostname, err := getHostname()
	if err == nil {
		info.Hostname = hostname
	}

	// 采集硬件 ID
	ids, err := collectHardwareIDs()
	if err != nil {
		return nil, fmt.Errorf("collect hardware IDs: %w", err)
	}

	// 拼接硬件 ID 并计算指纹
	raw := ""
	for _, id := range ids {
		raw += id + "|"
	}

	fp := sha256.Sum256([]byte(raw))
	info.Fingerprint = fmt.Sprintf("%x", fp)

	return info, nil
}
