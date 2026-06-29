// Package collector 负责从本地系统采集各种 OS 特征数据。
package collector

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"strings"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// rpmExecutor 是用于执行 rpm 命令的函数类型，便于测试时 mock。
type rpmExecutor func() (string, error)

// defaultRpmExecutor 是默认的 rpm 命令执行器。
func defaultRpmExecutor() (string, error) {
	cmd := exec.Command("rpm", "-qa", "--queryformat", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\\n")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", errors.New("failed to execute rpm command: " + err.Error())
	}

	return stdout.String(), nil
}

// CollectRPMPackages 采集当前系统已安装的所有 RPM 包信息。
// 该函数通过执行 rpm -qa 命令获取包列表。
//
// 返回值：
//   - []model.RPMPackage: 包列表
//   - error: 执行命令或解析失败时返回错误
func CollectRPMPackages() ([]model.RPMPackage, error) {
	return collectRPMPackagesWithExecutor(defaultRpmExecutor)
}

// collectRPMPackagesWithExecutor 使用给定的执行器采集 RPM 包信息。
// 该函数主要用于测试，允许注入 mock 的执行器。
func collectRPMPackagesWithExecutor(executor rpmExecutor) ([]model.RPMPackage, error) {
	output, err := executor()
	if err != nil {
		return nil, err
	}

	return parseRPMPackages(output)
}

// parseRPMPackages 解析 rpm 命令输出，生成 RPM 包列表。
func parseRPMPackages(output string) ([]model.RPMPackage, error) {
	lines := strings.Split(output, "\n")
	packages := make([]model.RPMPackage, 0, len(lines))

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			log.Printf("WARN: skipping malformed RPM line %d: %q (expected 4 fields, got %d)\n", lineNum+1, line, len(fields))
			continue
		}

		pkg := model.RPMPackage{
			Name:    fields[0],
			Version: fields[1],
			Release: normalizeRPMRelease(fields[2]),
			Arch:    fields[3],
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// normalizeRPMRelease 去掉 RPM release 中的 OS 发行版后缀。
// 例如 "5.oe2403sp3" 和 "5.oe2503" 都会归一化为 "5"，
// 以便比较同一包在不同 OS 版本中的差异时忽略 OS 标签。
func normalizeRPMRelease(release string) string {
	if idx := strings.Index(release, "."); idx != -1 {
		return release[:idx]
	}
	return release
}
