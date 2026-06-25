// Package cmd 定义了 os-compat-analyzer 命令行工具的命令结构。
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "os-compat-analyzer",
	Short: "OS 兼容性检查工具 - 采集与对比 Linux 系统特征",
	Long: `os-compat-analyzer 是一个用于检查 Linux OS 兼容性的命令行诊断工具。

功能：
  - collect: 采集当前系统的 OS 特征快照
  - report:  对比两个快照并生成 HTML 兼容性报告

示例：
  os-compat-analyzer collect -o os_a.json
  os-compat-analyzer report os_a.json os_b.json -o report.html`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). Only need to call once.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.os-compat-analyzer.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
