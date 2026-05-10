// Package cmd 定义了 os-checker 命令行工具的命令结构。
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/os-checker/internal/collector"
	"github.com/yourorg/os-checker/internal/model"
)

var (
	outputFile string
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "采集当前系统的 OS 特征快照",
	Long: `采集当前 Linux 系统的内核态与用户态接口特征，生成标准化的 JSON 数据。

采集内容包括：
  - 系统调用列表
  - 内核导出符号及 CRC
  - 用户态动态库符号
  - RPM 包列表

示例：
  os-checker collect -o os_a.json`,
	RunE: runCollect,
}

func init() {
	rootCmd.AddCommand(collectCmd)

	// Here you will define your flags and configuration settings.
	collectCmd.Flags().StringVarP(&outputFile, "output", "o", "", "输出文件路径 (默认输出到 stdout)")
	collectCmd.MarkFlagRequired("output")
}

// runCollect 执行数据采集
func runCollect(cmd *cobra.Command, args []string) error {
	log.Println("开始采集 OS 特征数据...")

	// 创建快照结构
	snapshot := &model.OSSnapshot{
		Metadata: collectMetadata(),
	}

	// 并发采集各类数据
	type collectResult struct {
		name  string
		data  interface{}
		err   error
	}

	resultChan := make(chan collectResult, 4)

	// 1. 采集 RPM 包
	go func() {
		pkgs, err := collector.CollectRPMPackages()
		resultChan <- collectResult{"RPM", pkgs, err}
	}()

	// 2. 采集用户态符号
	go func() {
		symbols, err := collector.CollectUserspaceSymbols(nil)
		resultChan <- collectResult{"UserspaceSymbols", symbols, err}
	}()

	// 3. 采集系统调用 (可能需要 root 权限)
	go func() {
		syscalls, err := collector.CollectSyscalls()
		resultChan <- collectResult{"Syscalls", syscalls, err}
	}()

	// 4. 采集内核符号 (可能需要 root 权限)
	go func() {
		symbols, err := collector.CollectKernelSymbols("")
		resultChan <- collectResult{"KernelSymbols", symbols, err}
	}()

	// 收集结果
	completed := 0
	for result := range resultChan {
		completed++

		switch result.name {
		case "RPM":
			if result.err != nil {
				log.Printf("WARN: RPM 包采集失败: %v (将继续使用空列表)", result.err)
				snapshot.RPMPackages = []model.RPMPackage{}
			} else {
				snapshot.RPMPackages = result.data.([]model.RPMPackage)
				log.Printf("INFO: 采集到 %d 个 RPM 包", len(snapshot.RPMPackages))
			}

		case "UserspaceSymbols":
			if result.err != nil {
				log.Printf("WARN: 用户态符号采集失败: %v (将继续使用空列表)", result.err)
				snapshot.UserspaceSymbols = []model.UserspaceSymbol{}
			} else {
				snapshot.UserspaceSymbols = result.data.([]model.UserspaceSymbol)
				log.Printf("INFO: 采集到 %d 个用户态符号", len(snapshot.UserspaceSymbols))
			}

		case "Syscalls":
			if result.err != nil {
				log.Printf("WARN: 系统调用采集失败: %v (将继续使用空列表)", result.err)
				snapshot.Syscalls = []model.Syscall{}
				// 添加警告标记到元数据
				snapshot.Metadata.Name += " (非 Root 采集)"
			} else {
				snapshot.Syscalls = result.data.([]model.Syscall)
				log.Printf("INFO: 采集到 %d 个系统调用", len(snapshot.Syscalls))
			}

		case "KernelSymbols":
			if result.err != nil {
				log.Printf("WARN: 内核符号采集失败: %v (将继续使用空列表)", result.err)
				snapshot.KernelSymbols = []model.KernelSymbol{}
				// 添加警告标记到元数据
				snapshot.Metadata.Name += " (非 Root 采集)"
			} else {
				snapshot.KernelSymbols = result.data.([]model.KernelSymbol)
				log.Printf("INFO: 采集到 %d 个内核符号", len(snapshot.KernelSymbols))
			}
		}

		if completed == 4 {
			close(resultChan)
		}
	}

	// 输出结果
	return writeOutput(snapshot, outputFile)
}

// collectMetadata 收集 OS 元数据
func collectMetadata() model.OSMetadata {
	metadata := model.OSMetadata{
		CollectedAt: time.Now(),
	}

	// 获取 OS 名称
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := string(data)
		if idx := findKey(content, "PRETTY_NAME"); idx >= 0 {
			metadata.Name = extractValue(content[idx:])
		}
	}

	// 获取内核版本
	if data, err := os.ReadFile("/proc/version"); err == nil {
		metadata.Version = string(data)
	}

	// 获取架构
	cmd := exec.Command("uname", "-m")
	if out, err := cmd.Output(); err == nil {
		metadata.Architecture = string(out)
	}

	return metadata
}

// findKey 在文本中查找 key
func findKey(text, key string) int {
	for i := range text {
		if len(text)-i < len(key) {
			break
		}
		if text[i:i+len(key)] == key {
			return i
		}
	}
	return -1
}

// extractValue 从 key=value 行中提取值
func extractValue(line string) string {
	idx := -1
	for i, c := range line {
		if c == '=' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}
	value := line[idx+1:]
	// 去除引号
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	return value
}

// writeOutput 将快照写入文件或 stdout
func writeOutput(snapshot *model.OSSnapshot, outputFile string) error {
	// 格式化 JSON
	jsonData, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if outputFile == "" {
		// 输出到 stdout
		fmt.Println(string(jsonData))
		return nil
	}

	// 写入文件
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	log.Printf("数据已保存到: %s", outputFile)
	return nil
}
