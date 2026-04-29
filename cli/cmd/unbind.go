// Package cmd 提供 CLI 命令实现。
package cmd

import (
	"context"
	"fmt"

	"cerberus.dev/sdk"

	"github.com/spf13/cobra"
)

var unbindLicenseID string
var unbindOldFingerprint string

// unbindCmd 换绑命令
var unbindCmd = &cobra.Command{
	Use:   "unbind",
	Short: "换绑机器",
	Long: `解绑旧机器，允许绑定新机器。

有换绑次数限制，超限将暂停 License。`,
	Run: func(cmd *cobra.Command, args []string) {
		if serverURL == "" {
			outputError(fmt.Errorf("server URL is required"))
			return
		}
		if unbindLicenseID == "" {
			outputError(fmt.Errorf("license ID is required"))
			return
		}
		if unbindOldFingerprint == "" {
			outputError(fmt.Errorf("old fingerprint is required"))
			return
		}

		client := sdk.NewClient(serverURL)
		result, err := client.Unbind(context.Background(), unbindLicenseID, unbindOldFingerprint)
		if err != nil {
			outputError(err)
			return
		}

		outputResult(result)
	},
}

func init() {
	rootCmd.AddCommand(unbindCmd)
	unbindCmd.Flags().StringVarP(&unbindLicenseID, "license", "l", "", "License ID")
	unbindCmd.Flags().StringVarP(&unbindOldFingerprint, "old-fingerprint", "o", "", "要解绑的机器指纹")
	_ = unbindCmd.MarkFlagRequired("license")
	_ = unbindCmd.MarkFlagRequired("old-fingerprint")
}
