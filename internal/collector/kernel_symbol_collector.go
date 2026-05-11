// Package collector 负责从本地系统采集各种 OS 特征数据。
package collector

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yourorg/os-checker/internal/model"
)

// CollectKernelSymbols 采集内核导出符号及其 CRC 校验值。
// 该函数按优先级查找 Module.symvers 文件进行解析。
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

	return "", errors.New("Module.symvers not found: please ensure kernel headers are installed or run as root")
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
