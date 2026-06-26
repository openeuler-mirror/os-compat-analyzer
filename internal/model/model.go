// Package model 定义了 OS 兼容性检查工具的核心数据结构。
// 这些结构体作为前后端交互和模块间解耦的唯一契约。
package model

import (
	"encoding/json"
	"sort"
	"time"
)

// OSSnapshot 表示一个 Linux 操作系统的完整快照，包含内核态与用户态的接口特征。
type OSSnapshot struct {
	// Metadata OS 元数据信息
	Metadata OSMetadata `json:"metadata"`

	// Syscalls 系统调用列表
	Syscalls []Syscall `json:"syscalls"`

	// KernelSymbols 内核导出符号列表
	KernelSymbols []KernelSymbol `json:"kernelSymbols"`

	// UserspaceSymbols 用户态动态库符号列表
	UserspaceSymbols []UserspaceSymbol `json:"userspaceSymbols"`

	// RPMPackages RPM 包列表
	RPMPackages []RPMPackage `json:"rpmPackages"`
}

// OSMetadata 表示操作系统的元数据信息。
type OSMetadata struct {
	// Name 操作系统名称（如 "Red Hat Enterprise Linux"）
	Name string `json:"name"`

	// Version 内核版本（如 "3.10.0-1160.el7.x86_64"）
	Version string `json:"version"`

	// Architecture 系统架构（如 "x86_64", "aarch64"）
	Architecture string `json:"architecture"`

	// CollectedAt 采集时间
	CollectedAt time.Time `json:"collectedAt"`

	// User 执行采集的用户名
	User string `json:"user"`
}

// Syscall 表示一个系统调用。
type Syscall struct {
	// Number 系统调用编号
	Number int `json:"number"`

	// Name 系统调用名称（如 "read", "write"）
	Name string `json:"name"`
}

// KernelSymbol 表示一个内核导出符号。
type KernelSymbol struct {
	// Name 符号名称
	Name string `json:"name"`

	// Module 所属内核模块（如 "kernel.ko", "ext4.ko"）
	Module string `json:"module"`

	// CRC 符号的 CRC 校验值，用于检测符号结构体变更
	CRC string `json:"crc"`
}

// UserspaceSymbol 表示一个用户态动态库导出的符号。
// 这是内部使用的数据结构。
type UserspaceSymbol struct {
	// SoPath 动态库路径（如 "/lib64/libc.so.6"）
	SoPath string `json:"-"`

	// SymbolName 符号名称（如 "malloc", "pthread_create"）
	SymbolName string `json:"symbolName"`

	// SymbolVersion 符号版本（如 "@@GLIBC_2.17", "@GLIBC_2.14"）
	SymbolVersion string `json:"symbolVersion"`
}

// UserspaceSymbolGroup 表示按 .so 文件分组的符号，用于 JSON 输出。
// 这样可以避免每个 symbol 都重复 soPath，减少 JSON 文件大小。
type UserspaceSymbolGroup struct {
	// SoPath 动态库路径
	SoPath string `json:"soPath"`

	// Symbols 该 .so 文件导出的符号列表
	Symbols []UserspaceSymbol `json:"symbols"`
}

// MarshalJSON 实现自定义的 JSON 序列化方法。
// 将 UserspaceSymbols 按 soPath 分组输出，以减少 JSON 文件大小。
func (s OSSnapshot) MarshalJSON() ([]byte, error) {
	// 按 SoPath 分组
	groups := make(map[string][]UserspaceSymbol)
	for _, sym := range s.UserspaceSymbols {
		groups[sym.SoPath] = append(groups[sym.SoPath], sym)
	}

	// 转换为切片并排序（按 SoPath 排序保证输出顺序一致）
	var groupedSymbols []UserspaceSymbolGroup
	for soPath, symbols := range groups {
		// 对每个 so 的符号按名称排序
		sort.Slice(symbols, func(i, j int) bool {
			return symbols[i].SymbolName < symbols[j].SymbolName
		})
		groupedSymbols = append(groupedSymbols, UserspaceSymbolGroup{
			SoPath:  soPath,
			Symbols: symbols,
		})
	}

	// 按 SoPath 排序
	sort.Slice(groupedSymbols, func(i, j int) bool {
		return groupedSymbols[i].SoPath < groupedSymbols[j].SoPath
	})

	// 构建输出结构
	output := struct {
		Metadata         OSMetadata             `json:"metadata"`
		Syscalls         []Syscall              `json:"syscalls"`
		KernelSymbols    []KernelSymbol         `json:"kernelSymbols"`
		UserspaceSymbols []UserspaceSymbolGroup `json:"userspaceSymbols"`
		RPMPackages      []RPMPackage           `json:"rpmPackages"`
	}{
		Metadata:         s.Metadata,
		Syscalls:         s.Syscalls,
		KernelSymbols:    s.KernelSymbols,
		UserspaceSymbols: groupedSymbols,
		RPMPackages:      s.RPMPackages,
	}

	return json.Marshal(output)
}

// RPMPackage 表示一个 RPM 软件包。
type RPMPackage struct {
	// Name 包名称（如 "glibc", "openssl"）
	Name string `json:"name"`

	// Version 主版本号（如 "2.17"）
	Version string `json:"version"`

	// Release 发行版本号（如 "el7"）
	Release string `json:"release"`

	// Architecture 软件包架构（如 "x86_64", "noarch"）
	Arch string `json:"arch"`
}
