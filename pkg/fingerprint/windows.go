//go:build windows

// Package fingerprint 提供 Windows 平台的机器指纹采集功能。
//
// Windows 平台使用 PowerShell 命令采集硬件信息：
//   - CPU ProcessorId：处理器唯一标识
//   - 磁盘序列号：物理磁盘序列号
//   - 主板 UUID：主板唯一标识
//   - 网卡 MAC 地址：物理网卡的 MAC 地址
package fingerprint

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// getHostname 获取 Windows 主机名。
//
// 返回：
//   - string: 主机名
//   - error: 获取失败时返回错误
func getHostname() (string, error) {
	// 优先使用环境变量
	if hostname := os.Getenv("COMPUTERNAME"); hostname != "" {
		return hostname, nil
	}

	// 备用方案：使用 PowerShell
	cmd := exec.Command("powershell", "-Command", "$env:COMPUTERNAME")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// collectHardwareIDs 采集 Windows 硬件 ID。
//
// 返回：
//   - []string: 硬件 ID 列表
//   - error: 采集失败时返回错误
func collectHardwareIDs() ([]string, error) {
	var ids []string

	// CPU ProcessorId
	if out, err := runPowerShell("Get-CimInstance Win32_Processor | Select-Object -ExpandProperty ProcessorId"); err == nil {
		if val := strings.TrimSpace(out); val != "" {
			ids = append(ids, val)
		}
	}

	// 磁盘序列号
	if out, err := runPowerShell("Get-CimInstance Win32_DiskDrive | Select-Object -First 1 -ExpandProperty SerialNumber"); err == nil {
		if val := strings.TrimSpace(out); val != "" {
			ids = append(ids, val)
		}
	}

	// 主板 UUID
	if out, err := runPowerShell("Get-CimInstance Win32_ComputerSystemProduct | Select-Object -ExpandProperty UUID"); err == nil {
		if val := strings.TrimSpace(out); val != "" {
			ids = append(ids, val)
		}
	}

	// 物理 NIC MAC 地址
	if out, err := runPowerShell("Get-NetAdapter -Physical | Where-Object {$_.Status -eq 'Up'} | Select-Object -First 1 -ExpandProperty MacAddress"); err == nil {
		if val := strings.TrimSpace(out); val != "" {
			ids = append(ids, val)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no hardware IDs collected")
	}

	return ids, nil
}

// runPowerShell 执行 PowerShell 命令并返回输出
func runPowerShell(command string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	cmd.SysProcAttr = nil // 使用默认设置

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// PowerShell 输出通常是 UTF-16 LE，需要转换
	// 尝试检测并转换编码
	result := string(out)

	// 如果包含乱码，尝试从 GBK 转换
	if strings.Contains(result, "�") || hasGarbledChars(result) {
		// 尝试 GBK 解码
		decoder := simplifiedchinese.GBK.NewDecoder()
		decoded, err := decoder.Bytes(out)
		if err == nil {
			return string(decoded), nil
		}

		// 尝试 UTF-16 LE 解码
		if len(out) >= 2 && out[0] == 0xFF && out[1] == 0xFE {
			// BOM 标记，是 UTF-16 LE
			reader := transform.NewReader(bytes.NewReader(out[2:]), simplifiedchinese.GBK.NewDecoder())
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(reader); err == nil {
				return buf.String(), nil
			}
		}
	}

	return result, nil
}

// hasGarbledChars 检查是否包含乱码字符
func hasGarbledChars(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return true
		}
	}
	return false
}
