// Package cmd 定义了 os-compat-analyzer 命令行工具的命令结构。
package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"atomgit.com/openeuler/os-compat-analyzer/internal/differ"
	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
	"github.com/spf13/cobra"
)

//go:embed templates/report.html
var embeddedTemplate string

var (
	outputHTML string
)

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "对比两个 OS 快照并生成 HTML 报告",
	Long: `读取两个由 collect 命令生成的 JSON 快照文件，进行差异对比，并生成可视化 HTML 报告。

示例：
  os-compat-analyzer report os_a.json os_b.json -o report.html`,
	Args: cobra.ExactArgs(2),
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&outputHTML, "output", "o", "report.html", "输出 HTML 文件路径")
}

// runReport 执行报告生成
func runReport(cmd *cobra.Command, args []string) error {
	log.Printf("正在加载快照文件: %s 和 %s", args[0], args[1])

	// 读取两个快照文件
	osA, err := loadSnapshot(args[0])
	if err != nil {
		return fmt.Errorf("failed to load snapshot A: %w", err)
	}

	osB, err := loadSnapshot(args[1])
	if err != nil {
		return fmt.Errorf("failed to load snapshot B: %w", err)
	}

	log.Println("正在计算差异...")

	// 计算差异
	diffResult := differ.Compare(osA, osB)

	log.Printf("差异计算完成:")
	log.Printf("  - Syscalls: A独有 %d, B独有 %d",
		len(diffResult.SyscallsDiff.OnlyInA), len(diffResult.SyscallsDiff.OnlyInB))
	log.Printf("  - Kernel Symbols: A独有 %d, B独有 %d, CRC冲突 %d",
		len(diffResult.KernelSymbolsDiff.OnlyInA),
		len(diffResult.KernelSymbolsDiff.OnlyInB),
		len(diffResult.KernelSymbolsDiff.Modified))
	log.Printf("  - RPM Packages: A独有 %d, B独有 %d, 版本变化 %d",
		len(diffResult.RPMPackagesDiff.OnlyInA),
		len(diffResult.RPMPackagesDiff.OnlyInB),
		len(diffResult.RPMPackagesDiff.Modified))

	// 生成 HTML 报告
	if err := generateHTMLReport(diffResult, osA, osB, outputHTML); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}

	log.Printf("报告已生成: %s", outputHTML)
	return nil
}

// loadSnapshot 从文件加载 OSSnapshot
func loadSnapshot(path string) (*model.OSSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var snapshot model.OSSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// generateHTMLReport 生成 HTML 报告
func generateHTMLReport(diffResult *differ.DiffResult, osA, osB *model.OSSnapshot, outputPath string) error {
	// 加载模板
	templateContent := loadTemplate()

	// 准备完整的 JSON 数据（包含 diffResult 和 OS 信息）
	fullData := map[string]interface{}{
		"diffResult": diffResult,
		"OS_A":       osA,
		"OS_B":       osB,
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(fullData)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// 替换占位符
	result := replacePlaceholder(templateContent, string(jsonData))

	// 写入文件
	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return nil
}

// replacePlaceholder 替换 HTML 中的占位符为 JSON 数据
func replacePlaceholder(html, jsonData string) string {
	// 替换 INJECT_DATA_HERE 占位符
	html = replaceOnce(html, "<!-- INJECT_DATA_HERE -->", jsonData)

	// 同时兼容 window.__INITIAL_STATE__ 方式
	stateScript := fmt.Sprintf("window.__INITIAL_STATE__ = %s;", jsonData)
	html = replaceOnce(html, "/* INJECT_STATE_HERE */", stateScript)

	return html
}

// replaceOnce 替换字符串中的第一个匹配项
func replaceOnce(s, old, new string) string {
	if old == "" {
		return s
	}
	// 直接查找占位符并替换
	if idx := findPlaceholder(s, old); idx >= 0 {
		return s[:idx] + new + s[idx+len(old):]
	}
	return s
}

// findPlaceholder 查找占位符的位置
func findPlaceholder(s, placeholder string) int {
	for i := 0; i <= len(s)-len(placeholder); i++ {
		if s[i:i+len(placeholder)] == placeholder {
			return i
		}
	}
	return -1
}

// loadTemplate 尝试读取模板文件，优先使用嵌入的模板
func loadTemplate() string {
	// 首先尝试使用嵌入的模板
	if embeddedTemplate != "" {
		return embeddedTemplate
	}

	// 尝试读取文件系统中的模板
	possiblePaths := []string{
		"cmd/templates/report.html",
		"web/dist/index.html",
		"dist/index.html",
	}

	for _, path := range possiblePaths {
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}

	// 返回默认模板
	return getDefaultTemplate()
}

// getDefaultTemplate 返回默认的内置模板
func getDefaultTemplate() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OS 兼容性报告</title>
    /* INJECT_STATE_HERE */
</head>
<body>
    <div id="app">兼容性报告加载中...</div>
    <script>
        // 从 script 标签中读取数据
        const dataElement = document.getElementById('data');
        if (dataElement && dataElement.textContent) {
            try {
                const diffData = JSON.parse(dataElement.textContent);
                window.__INITIAL_STATE__ = diffData;
            } catch(e) {
                console.error('Failed to parse data:', e);
            }
        }
    </script>
    <script id="data" type="application/json">
    <!-- INJECT_DATA_HERE -->
    </script>
</body>
</html>`
}
