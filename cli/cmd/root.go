// Package cmd 提供 CLI 命令实现。
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	serverURL       string
	fingerprintFlag string
	outputFormat    string
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "cerberus-client",
	Short: "Cerberus License 验证客户端",
	Long: `Cerberus License 验证客户端工具。

支持在线验证，适用于 Python/Java/Go 等多语言集成。
所有命令输出 JSON 格式，便于程序解析。

使用示例：
  cerberus-client fingerprint                          # 采集机器指纹
  cerberus-client activate --license <id> --server <url>  # 激活
  cerberus-client verify --license <id> --server <url>    # 在线验证
  cerberus-client heartbeat --license <id> --server <url> # 心跳
  cerberus-client unbind --license <id> --server <url>    # 换绑

环境变量：
  CERBERUS_SERVER_URL - 默认服务器地址`,
}

// Execute 执行命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 全局 flag
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", os.Getenv("CERBERUS_SERVER_URL"), "Cerberus Server URL")
	rootCmd.PersistentFlags().StringVarP(&fingerprintFlag, "fingerprint", "f", "auto", "机器指纹 (auto=自动采集)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "输出格式 (json|text)")
}
