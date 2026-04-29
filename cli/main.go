// Package main 提供 cerberus-client CLI 工具。
//
// 支持：
//   - 在线验证：通过 HTTP 调用 Cerberus Server
//
// 使用示例：
//
//	cerberus-client fingerprint                          # 采集机器指纹
//	cerberus-client activate --license <id> --server <url>  # 激活机器
//	cerberus-client verify --license <id> --server <url>    # 在线验证
//	cerberus-client heartbeat --license <id> --server <url> # 心跳
//	cerberus-client unbind --license <id> --server <url>    # 自助换绑
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"cerberus.dev/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// 输出错误为 JSON
		result := map[string]interface{}{
			"valid":   false,
			"reason":  err.Error(),
			"success": false,
			"error":   err.Error(),
		}
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
		os.Exit(1)
	}
}
