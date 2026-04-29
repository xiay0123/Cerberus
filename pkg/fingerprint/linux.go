//go:build linux

// Package fingerprint 提供 Linux 平台的机器指纹采集功能。
//
// Linux 平台通过读取文件系统和执行命令采集硬件信息：
//   - Machine ID：/etc/machine-id（systemd 机器标识）
//   - CPU 信息：/proc/cpuinfo（CPU 型号和物理 ID）
//   - 磁盘序列号：使用 lsblk 命令
//   - MAC 地址：/sys/class/net/*/address（物理网卡）
package fingerprint

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// getHostname 获取 Linux 主机名。
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

// collectHardwareIDs 采集 Linux 硬件 ID。
//
// 采集项目：
//  1. Machine ID - 读取 /etc/machine-id
//  2. CPU 信息 - 读取 /proc/cpuinfo（model name、physical id）
//  3. 磁盘序列号 - 使用 lsblk 命令
//  4. MAC 地址 - 读取 /sys/class/net/*/address（排除 lo 回环接口）
//
// 返回：
//   - []string: 硬件 ID 列表
//   - error: 采集失败时返回错误
func collectHardwareIDs() ([]string, error) {
	var ids []string

	// machine-id（systemd 机器标识）
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		ids = append(ids, strings.TrimSpace(string(data)))
	}

	// CPU info（读取 CPU 型号和物理 ID）
	if out, err := exec.Command("cat", "/proc/cpuinfo").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "physical id") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					ids = append(ids, strings.TrimSpace(parts[1]))
				}
				break
			}
		}
	}

	// Root disk serial（磁盘序列号）
	if out, err := exec.Command("lsblk", "--serial", "-n", "-d", "-o", "SERIAL").Output(); err == nil {
		serial := strings.TrimSpace(string(out))
		if serial != "" {
			ids = append(ids, serial)
		}
	}

	// MAC addresses of physical NICs（物理网卡 MAC 地址）
	if entries, err := os.ReadDir("/sys/class/net"); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if name == "lo" {
				continue // 跳过回环接口
			}
			if data, err := os.ReadFile("/sys/class/net/" + name + "/address"); err == nil {
				mac := strings.TrimSpace(string(data))
				if mac != "" && mac != "00:00:00:00:00:00" {
					ids = append(ids, mac)
				}
			}
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no hardware IDs collected")
	}

	return ids, nil
}
