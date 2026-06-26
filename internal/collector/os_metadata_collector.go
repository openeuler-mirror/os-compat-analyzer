// Package collector 负责从本地系统采集各种 OS 特征数据。
package collector

import (
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// CollectOSMetadata 收集当前系统的 OS 元数据。
//
// 返回值：
//   - model.OSMetadata: 包含 OS 名称、内核版本、架构和采集时间
func CollectOSMetadata() model.OSMetadata {
	// 获取 OS 名称
	var osRelease string
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		osRelease = string(data)
	}

	// 获取内核版本
	var kernelVersion string
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		kernelVersion = string(out)
	}

	// 获取架构
	var architecture string
	if out, err := exec.Command("uname", "-m").Output(); err == nil {
		architecture = string(out)
	}

	// 获取当前用户
	var currentUser string
	if u, err := user.Current(); err == nil && u.Username != "" {
		currentUser = u.Username
	} else {
		currentUser = os.Getenv("USER")
	}

	return collectOSMetadataWithInputs(osRelease, kernelVersion, architecture, currentUser)
}

// collectOSMetadataWithInputs 根据给定的原始输入构建 OS 元数据。
// 该函数主要用于测试，允许注入 mock 的输入数据。
func collectOSMetadataWithInputs(osRelease, kernelVersion, architecture, currentUser string) model.OSMetadata {
	metadata := model.OSMetadata{
		CollectedAt: time.Now(),
	}

	// 获取 OS 名称
	if idx := findKey(osRelease, "PRETTY_NAME"); idx >= 0 {
		metadata.Name = extractValue(osRelease[idx:], "PRETTY_NAME")
	}

	// 获取内核版本
	metadata.Version = strings.TrimSpace(kernelVersion)

	// 获取架构
	metadata.Architecture = strings.TrimSpace(architecture)

	// 获取当前用户
	metadata.User = strings.TrimSpace(currentUser)

	return metadata
}

// findKey 在文本中查找 key，要求 key 后紧跟 '='。
func findKey(text, key string) int {
	if key == "" {
		return 0
	}

	target := key + "="
	for i := range text {
		if len(text)-i < len(target) {
			break
		}
		if text[i:i+len(target)] == target {
			return i
		}
	}
	return -1
}

// extractValue 从 text 中 key=value 配对提取值。
// text 必须以 key 开头；只取到当前行结束，并去除首尾的引号。
func extractValue(text, key string) string {
	if key == "" {
		return ""
	}
	if !strings.HasPrefix(text, key) {
		return ""
	}

	// 跳过 key 和 '='
	value := text[len(key):]
	value = strings.TrimLeftFunc(value, func(r rune) bool {
		return r == '=' || r == ' ' || r == '\t'
	})

	// 只取当前行
	for i, c := range value {
		if c == '\n' {
			value = value[:i]
			break
		}
	}

	// 去除引号
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	return value
}
