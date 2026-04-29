//go:build darwin

// Package fingerprint 提供 macOS 平台的机器指纹采集功能。
//
// macOS 平台使用系统命令采集硬件信息：
//   - IOPlatformUUID：平台唯一标识（通过 ioreg 命令）
//   - CPU 品牌：处理器型号（通过 sysctl 命令）
//   - 磁盘序列号：通过 ioreg 命令
//   - MAC 地址：en0 网卡的 MAC 地址（通过 ifconfig 命令）
package fingerprint

import (
	"fmt"
	"os/exec"
	"strings"
)

// getHostname 获取 macOS 主机名。
//
// 使用 hostname 命令获取当前主机名称。
//
// 返回：
//   - string: 主机名
//   - error: 获取失败时返回错误
func getHostname() (string, error) {
	cmd := exec.Command("hostname")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// collectHardwareIDs 采集 macOS 硬件 ID。
//
// 采集项目：
//  1. IOPlatformUUID - 使用 ioreg 获取平台唯一标识
//  2. CPU 品牌 - 使用 sysctl 获取处理器型号
//  3. 磁盘序列号 - 使用 ioreg 获取
//  4. MAC 地址 - 使用 ifconfig 获取 en0 网卡的 MAC 地址
//
// 返回：
//   - []string: 硬件 ID 列表
//   - error: 采集失败时返回错误
func collectHardwareIDs() ([]string, error) {
	var ids []string

	// IOPlatformUUID（平台唯一标识）
	if out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "IOPlatformUUID") {
				parts := strings.Split(line, `"`)
				if len(parts) >= 4 {
					ids = append(ids, parts[3])
				}
				break
			}
		}
	}

	// CPU brand（处理器型号）
	if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		ids = append(ids, strings.TrimSpace(string(out)))
	}

	// Disk serial（磁盘序列号）
	if out, err := exec.Command("ioreg", "-rd1", "-c", "IOAHCIBlockStorageDevice").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "Serial Number") {
				parts := strings.Split(line, `"`)
				if len(parts) >= 4 {
					ids = append(ids, parts[3])
				}
				break
			}
		}
	}

	// MAC address of en0（默认网卡 MAC 地址）
	if out, err := exec.Command("ifconfig", "en0").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "ether") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					ids = append(ids, parts[1])
				}
				break
			}
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no hardware IDs collected")
	}

	return ids, nil
}
