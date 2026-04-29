// Package cmd 提供 CLI 命令实现。
package cmd

import (
	"encoding/json"
	"fmt"

	"cerberus.dev/pkg/fingerprint"

	"github.com/spf13/cobra"
)

// fingerprintCmd 采集机器指纹命令
var fingerprintCmd = &cobra.Command{
	Use:   "fingerprint",
	Short: "采集机器指纹",
	Long: `采集当前机器的硬件指纹信息。

指纹包含：CPU ID、磁盘序列号、主板 UUID、MAC 地址等。
输出 JSON 格式的机器信息。`,
	Run: func(cmd *cobra.Command, args []string) {
		info, err := fingerprint.Collect()
		if err != nil {
			outputError(err)
			return
		}

		outputResult(info)
	},
}

func init() {
	rootCmd.AddCommand(fingerprintCmd)
}

func outputResult(data interface{}) {
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(jsonData))
}

func outputError(err error) {
	result := map[string]interface{}{
		"success": false,
		"error":   err.Error(),
	}
	jsonData, _ := json.Marshal(result)
	fmt.Println(string(jsonData))
}
