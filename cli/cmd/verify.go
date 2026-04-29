// Package cmd 提供 CLI 命令实现。
package cmd

import (
	"context"
	"fmt"

	"cerberus.dev/sdk"

	"github.com/spf13/cobra"
)

var verifyLicenseID string
var verifyPublicIP bool

// verifyCmd 验证命令
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "验证 License",
	Long: `在线验证 License 有效性。

验证流程：
  1. 采集机器指纹（自动或手动指定）
  2. 向 Cerberus Server 发送验证请求
  3. 返回验证结果（有效期、产品信息等）

安全机制：
  - 验证时会检查机器是否已激活
  - 如果启用了 IP 绑定，会验证 IP 地址`,
	Run: func(cmd *cobra.Command, args []string) {
		if verifyLicenseID == "" {
			outputError(fmt.Errorf("license ID is required"))
			return
		}
		if serverURL == "" {
			outputError(fmt.Errorf("server URL is required"))
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
		if verifyPublicIP {
			opts = append(opts, sdk.WithPublicIP())
		}

		client := sdk.NewClient(serverURL)
		result, err := client.Verify(context.Background(), verifyLicenseID, opts...)
		if err != nil {
			outputError(err)
			return
		}

		outputResult(result)
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().StringVarP(&verifyLicenseID, "license", "l", "", "License ID")
	verifyCmd.Flags().BoolVar(&verifyPublicIP, "public-ip", false, "获取公网 IP")
	_ = verifyCmd.MarkFlagRequired("license")
}
