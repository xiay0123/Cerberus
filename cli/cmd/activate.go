// Package cmd 提供 CLI 命令实现。
package cmd

import (
	"context"
	"fmt"

	"cerberus.dev/sdk"

	"github.com/spf13/cobra"
)

var activateLicenseID string
var activatePublicIP bool

// activateCmd 激活命令
var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "激活 License",
	Long: `激活 License，将当前机器绑定到 License。

激活流程：
  1. 采集机器指纹（CPU ID、磁盘序列号、主板 UUID、MAC 地址等）
  2. 向 Cerberus Server 发送激活请求
  3. 服务端记录机器信息并返回激活结果

注意事项：
  - 每个 License 有最大机器数限制
  - 如果已达到上限，需要先换绑或管理员解绑`,
	Run: func(cmd *cobra.Command, args []string) {
		if serverURL == "" {
			outputError(fmt.Errorf("server URL is required (use --server or CERBERUS_SERVER_URL)"))
			return
		}
		if activateLicenseID == "" {
			outputError(fmt.Errorf("license ID is required (use --license)"))
			return
		}

		// 采集指纹
		var opts []sdk.Option
		if fingerprintFlag == "auto" {
			opts = append(opts, sdk.WithFingerprintAuto())
		} else {
			opts = append(opts, sdk.WithFingerprint(fingerprintFlag))
		}

		// 获取公网 IP
		if activatePublicIP {
			opts = append(opts, sdk.WithPublicIP())
		}

		// 激活
		client := sdk.NewClient(serverURL)
		resp, err := client.Activate(context.Background(), activateLicenseID, opts...)
		if err != nil {
			outputError(err)
			return
		}

		outputResult(resp)
	},
}

func init() {
	rootCmd.AddCommand(activateCmd)
	activateCmd.Flags().StringVarP(&activateLicenseID, "license", "l", "", "License ID")
	activateCmd.Flags().BoolVar(&activatePublicIP, "public-ip", false, "获取公网 IP")
	_ = activateCmd.MarkFlagRequired("license")
}
