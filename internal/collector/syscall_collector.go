// Package collector 负责从本地系统采集各种 OS 特征数据。
package collector

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yourorg/os-checker/internal/model"
)

// CollectSyscalls 采集当前内核支持的所有系统调用。
// 该函数读取 /proc/kallsyms 文件获取系统调用信息。
//
// 返回值：
//   - []model.Syscall: 系统调用列表
//   - error: 读取失败时返回错误
func CollectSyscalls() ([]model.Syscall, error) {
	// 尝试读取 /proc/kallsyms
	f, err := os.Open("/proc/kallsyms")
	if err != nil {
		return nil, handleKallsymsError(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	syscalls := make([]model.Syscall, 0)
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// 格式: address type symbol
		if len(fields) < 3 {
			continue
		}

		_ = fields[0] // address，当前不需要使用
		symType := fields[1]
		symbol := fields[2]

		// 过滤条件:
		// 1. 符号以 sys_ 开头
		// 2. 类型为小写字母 't' (文本段内的符号)
		// 3. 不是 sys_ni_syscall (not implemented)
		if !strings.HasPrefix(symbol, "sys_") {
			continue
		}

		// 类型必须是 't' (小写，表示文本段内的符号)
		if symType != "t" {
			continue
		}

		// 跳过 sys_ni_syscall (not implemented)
		if symbol == "sys_ni_syscall" {
			continue
		}

		// 去除前缀获取系统调用名
		syscallName := strings.TrimPrefix(symbol, "sys_")

		// 去重（同一符号可能有多个地址）
		if seen[syscallName] {
			continue
		}
		seen[syscallName] = true

		syscalls = append(syscalls, model.Syscall{
			Number: 0, // TODO: 可通过架构特定表映射编号
			Name:   syscallName,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /proc/kallsyms: %w", err)
	}

	// 按名称排序，保证输出顺序一致
	sortSyscalls(syscalls)

	return syscalls, nil
}

// handleKallsymsError 处理 /proc/kallsyms 读取错误
func handleKallsymsError(err error) error {
	if os.IsNotExist(err) {
		return errors.New("/proc/kallsyms not found: this may happen in containers or non-Linux systems")
	}
	if os.IsPermission(err) {
		return errors.New("permission denied reading /proc/kallsyms: root privileges required (non-root users can only see symbols from their own process)")
	}
	return fmt.Errorf("failed to open /proc/kallsyms: %w", err)
}

// sortSyscalls 按名称排序系统调用列表
func sortSyscalls(syscalls []model.Syscall) {
	sort.Slice(syscalls, func(i, j int) bool {
		return syscalls[i].Name < syscalls[j].Name
	})
}
