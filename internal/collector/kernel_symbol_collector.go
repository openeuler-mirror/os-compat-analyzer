// Package collector 负责从本地系统采集各种 OS 特征数据。
package collector

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// CollectKernelSymbols 采集内核导出符号及其 CRC 校验值。
// 该函数按优先级查找 Module.symvers 文件进行解析。
// 当 Module.symvers 不可用时，回退到 /proc/kallsyms（无 CRC 值）。
//
// 参数：
//   - unameRelease: 内核版本号（如 "3.10.0-1160.el7.x86_64"）
//
// 返回值：
//   - []model.KernelSymbol: 内核符号列表
//   - error: 查找或解析失败时返回错误
func CollectKernelSymbols(unameRelease string) ([]model.KernelSymbol, error) {
	if unameRelease == "" {
		// 如果未提供，尝试获取当前内核版本
		var err error
		unameRelease, err = getKernelRelease()
		if err != nil {
			return nil, fmt.Errorf("failed to get kernel release: %w", err)
		}
	}

	// 按优先级查找 symvers 文件
	symversPath, err := findSymversFile(unameRelease)
	if err != nil {
		// Module.symvers 不可用，尝试 /proc/kallsyms 作为备选
		kallsymsPath := "/proc/kallsyms"
		if _, statErr := os.Stat(kallsymsPath); statErr == nil {
			symbols, parseErr := parseKallsymsFile(kallsymsPath)
			if parseErr != nil {
				return nil, fmt.Errorf("Module.symvers not found (%w) and failed to parse /proc/kallsyms: %v", err, parseErr)
			}
			if len(symbols) == 0 {
				return nil, fmt.Errorf("Module.symvers not found and /proc/kallsyms yielded no symbols; note: /proc/kallsyms may show zero addresses without root, try running as root")
			}
			return symbols, nil
		}
		return nil, err
	}

	// 解析文件
	symbols, err := parseSymversFile(symversPath)
	if err != nil {
		return nil, err
	}

	return symbols, nil
}

// getKernelRelease 获取当前内核版本
func getKernelRelease() (string, error) {
	cmd := exec.Command("uname", "-r")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run uname -r: %w", err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// findSymversFile 查找 Module.symvers 文件路径
func findSymversFile(unameRelease string) (string, error) {
	// 优先级 1: /lib/modules/{unameRelease}/build/Module.symvers
	buildSymvers := filepath.Join("/lib/modules", unameRelease, "build", "Module.symvers")
	if _, err := os.Stat(buildSymvers); err == nil {
		return buildSymvers, nil
	}

	// 优先级 2: /boot/symvers-{unameRelease}.gz
	bootSymversGz := filepath.Join("/boot", fmt.Sprintf("symvers-%s.gz", unameRelease))
	if _, err := os.Stat(bootSymversGz); err == nil {
		return bootSymversGz, nil
	}

	// 优先级 3: /boot/symvers-{unameRelease} (非压缩)
	bootSymvers := filepath.Join("/boot", fmt.Sprintf("symvers-%s", unameRelease))
	if _, err := os.Stat(bootSymvers); err == nil {
		return bootSymvers, nil
	}

	return "", fmt.Errorf("Module.symvers not found for kernel %s: install kernel-devel (%s-devel) or kernel-headers package, or run as root", unameRelease, unameRelease)
}

// parseSymversFile 解析 Module.symvers 文件
// 格式: CRC\tSymbolName\tModule\tExportType
func parseSymversFile(path string) ([]model.KernelSymbol, error) {
	var scanner *bufio.Scanner

	// 判断是否为 gzip 压缩文件
	if strings.HasSuffix(path, ".gz") {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer f.Close()

		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		scanner = bufio.NewScanner(gzReader)
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer f.Close()

		scanner = bufio.NewScanner(f)
	}

	symbols := make([]model.KernelSymbol, 0)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析行: CRC<tab>SymbolName<tab>Module<tab>ExportType
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}

		crc := fields[0]
		symbolName := fields[1]
		module := fields[2]

		// 跳过空符号名
		if symbolName == "" {
			continue
		}

		symbols = append(symbols, model.KernelSymbol{
			Name:   symbolName,
			Module: module,
			CRC:    crc,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return symbols, nil
}

// parseKallsymsFile 解析 /proc/kallsyms 文件作为 Module.symvers 不可用时的备选方案。
// 格式: address type name [module]
// 注意: kallsyms 不提供 CRC 值，CRC 字段将为空字符串。
func parseKallsymsFile(path string) ([]model.KernelSymbol, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	symbols := make([]model.KernelSymbol, 0)
	scanner := bufio.NewScanner(f)
	// kallsyms 行数较多，增大 buffer
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 格式: address type name [module]
		// 使用 Fields 按空白符分割
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// fields[0] = address, fields[1] = type, fields[2] = name
		symbolName := fields[2]

		// 跳过编译器生成的噪音符号（ARM mapping symbols、局部标签等）
		if strings.HasPrefix(symbolName, "$") || strings.HasPrefix(symbolName, ".") {
			continue
		}

		// 确定模块名：有 [module] 字段则提取，否则为 vmlinux（内置符号）
		module := "vmlinux"
		if len(fields) >= 4 {
			modField := fields[3]
			if strings.HasPrefix(modField, "[") && strings.HasSuffix(modField, "]") {
				module = modField[1 : len(modField)-1]
			}
		}

		symbols = append(symbols, model.KernelSymbol{
			Name:   symbolName,
			Module: module,
			CRC:    "", // kallsyms 不提供 CRC
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return symbols, nil
}
