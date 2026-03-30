// Package main 提供 icooclaw 的程序入口。
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"icooclaw/pkg/app"
)

var (
	cfgFile string
	version = "dev"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:          "icooclaw",
	Short:        "启动 icooclaw 网关服务",
	Long:         `icooclaw is an AI agent framework with multi-channel support.`,
	Example:      "  icooclaw -c config.toml\n  icooclaw version",
	SilenceUsage: true,
	Run:          runGateway,
}

var startCmd = &cobra.Command{
	Use:        "start",
	Short:      "已废弃: 启动网关服务",
	Hidden:     true,
	Deprecated: "请直接使用 `icooclaw -c config.toml`",
	Run:        runGateway,
}

var gatewayCmd = &cobra.Command{
	Use:        "gateway",
	Short:      "已废弃: 启动网关服务",
	Hidden:     true,
	Deprecated: "请直接使用 `icooclaw -c config.toml`",
	Run:        runGateway,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("icooclaw version:", version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.toml)")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(versionCmd)
}

// runGateway 启动网关服务
func runGateway(cmd *cobra.Command, args []string) {
	// 创建应用实例
	app := app.NewApp()
	// 初始化
	if err := app.Init(cfgFile); err != nil {
		slog.Error("初始化失败", "error", err)
		os.Exit(1)
	}
	// 运行网关服务
	app.RunGateway()

	slog.Info("网关服务已停止")
}
