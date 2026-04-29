// Package cmd 提供 CLI 命令实现。
package cmd

import (
	"context"
	"fmt"

	"cerberus.dev/sdk"

	"github.com/spf13/cobra"
)

var heartbeatLicenseID string
var heartbeatPublicIP bool

// heartbeatCmd 心跳命令
var heartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "发送心跳",
	Long: `发送心跳到 Cerberus Server。

心跳用于更新机器的 last-seen 时间和 IP 地址。`,
	Run: func(cmd *cobra.Command, args []string) {
		if serverURL == "" {
			outputError(fmt.Errorf("server URL is required"))
			return
		}
		if heartbeatLicenseID == "" {
			outputError(fmt.Errorf("license ID is required"))
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
		if heartbeatPublicIP {
			opts = append(opts, sdk.WithPublicIP())
		}

		client := sdk.NewClient(serverURL)
		if err := client.Heartbeat(context.Background(), heartbeatLicenseID, opts...); err != nil {
			outputError(err)
			return
		}

		outputResult(map[string]interface{}{
			"success": true,
			"message": "heartbeat received",
		})
	},
}

func init() {
	rootCmd.AddCommand(heartbeatCmd)
	heartbeatCmd.Flags().StringVarP(&heartbeatLicenseID, "license", "l", "", "License ID")
	heartbeatCmd.Flags().BoolVar(&heartbeatPublicIP, "public-ip", false, "获取公网 IP")
	_ = heartbeatCmd.MarkFlagRequired("license")
}
