// Package differ 实现了 OS 兼容性数据的差异比对引擎。
package differ

import (
	"sort"
	"strings"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// DiffResult 表示两个 OS 快照的差异结果
type DiffResult struct {
	// SyscallsDiff 系统调用差异
	SyscallsDiff *SyscallsDiff `json:"syscallsDiff"`

	// KernelSymbolsDiff 内核符号差异
	KernelSymbolsDiff *KernelSymbolsDiff `json:"kernelSymbolsDiff"`

	// UserspaceSymbolsDiff 用户态符号差异
	UserspaceSymbolsDiff *UserspaceSymbolsDiff `json:"userspaceSymbolsDiff"`

	// RPMPackagesDiff RPM 包差异
	RPMPackagesDiff *RPMPackagesDiff `json:"rpmPackagesDiff"`
}

// SyscallsDiff 系统调用差异
type SyscallsDiff struct {
	// OnlyInA 仅在 OS A 中存在的系统调用
	OnlyInA []model.Syscall `json:"onlyInA"`

	// OnlyInB 仅在 OS B 中存在的系统调用
	OnlyInB []model.Syscall `json:"onlyInB"`

	// TotalInA OS A 中的系统调用总数
	TotalInA int `json:"totalInA"`

	// TotalInB OS B 中的系统调用总数
	TotalInB int `json:"totalInB"`
}

// KernelSymbolsDiff 内核符号差异
type KernelSymbolsDiff struct {
	// OnlyInA 仅在 OS A 中存在的符号
	OnlyInA []model.KernelSymbol `json:"onlyInA"`

	// OnlyInB 仅在 OS B 中存在的符号
	OnlyInB []model.KernelSymbol `json:"onlyInB"`

	// Modified CRC 发生变化的符号（严重风险）
	Modified []KernelSymbolModified `json:"modified"`

	// TotalInA OS A 中的符号总数
	TotalInA int `json:"totalInA"`

	// TotalInB OS B 中的符号总数
	TotalInB int `json:"totalInB"`
}

// KernelSymbolModified 表示 CRC 发生变化的符号
type KernelSymbolModified struct {
	model.KernelSymbol
	// CRCInA OS A 中的 CRC
	CRCInA string `json:"crcInA"`
	// CRCInB OS B 中的 CRC
	CRCInB string `json:"crcInB"`
}

// UserspaceSymbolsDiff 用户态符号差异
type UserspaceSymbolsDiff struct {
	// 按 SoPath 分组的差异
	BySoPath map[string]*UserspaceSymbolGroupDiff `json:"bySoPath"`

	// 总计统计
	TotalInA int `json:"totalInA"`
	TotalInB int `json:"totalInB"`
}

// UserspaceSymbolGroupDiff 表示单个 .so 文件的符号差异
type UserspaceSymbolGroupDiff struct {
	// SoPath 动态库路径
	SoPath string `json:"soPath"`

	// OnlyInA 仅在 OS A 中存在的符号
	OnlyInA []model.UserspaceSymbol `json:"onlyInA"`

	// OnlyInB 仅在 OS B 中存在的符号
	OnlyInB []model.UserspaceSymbol `json:"onlyInB"`

	// Modified 版本发生变化的符号
	Modified []UserspaceSymbolModified `json:"modified"`

	// Common 在两个 OS 中版本相同的符号
	Common []model.UserspaceSymbol `json:"common"`
}

// UserspaceSymbolModified 表示版本发生变化的符号
type UserspaceSymbolModified struct {
	model.UserspaceSymbol
	// VersionInA OS A 中的版本
	VersionInA string `json:"versionInA"`
	// VersionInB OS B 中的版本
	VersionInB string `json:"versionInB"`
}

// RPMPackagesDiff RPM 包差异
type RPMPackagesDiff struct {
	// OnlyInA 仅在 OS A 中存在的包
	OnlyInA []model.RPMPackage `json:"onlyInA"`

	// OnlyInB 仅在 OS B 中存在的包
	OnlyInB []model.RPMPackage `json:"onlyInB"`

	// Modified 版本发生变化的包
	Modified []RPMPackageModified `json:"modified"`

	// TotalInA OS A 中的包总数
	TotalInA int `json:"totalInA"`

	// TotalInB OS B 中的包总数
	TotalInB int `json:"totalInB"`
}

// RPMPackageModified 表示版本发生变化的包
type RPMPackageModified struct {
	model.RPMPackage
	// VersionInA OS A 中的版本
	VersionInA string `json:"versionInA"`
	// VersionInB OS B 中的版本
	VersionInB string `json:"versionInB"`
	// Upgrade true 表示升级，false 表示降级
	Upgrade bool `json:"upgrade"`
}

// Compare 比较两个 OSSnapshot，返回差异结果
func Compare(osA, osB *model.OSSnapshot) *DiffResult {
	return &DiffResult{
		SyscallsDiff:         compareSyscalls(osA.Syscalls, osB.Syscalls),
		KernelSymbolsDiff:    compareKernelSymbols(osA.KernelSymbols, osB.KernelSymbols),
		UserspaceSymbolsDiff: compareUserspaceSymbols(osA.UserspaceSymbols, osB.UserspaceSymbols),
		RPMPackagesDiff:      compareRPMPackages(osA.RPMPackages, osB.RPMPackages),
	}
}

// compareSyscalls 比较系统调用
func compareSyscalls(a, b []model.Syscall) *SyscallsDiff {
	// 构建 map 以提高查找效率
	aMap := make(map[string]model.Syscall)
	bMap := make(map[string]model.Syscall)

	for _, s := range a {
		aMap[s.Name] = s
	}
	for _, s := range b {
		bMap[s.Name] = s
	}

	// 查找仅在 A 或 B 中存在的
	var onlyInA, onlyInB []model.Syscall
	for name, s := range aMap {
		if _, exists := bMap[name]; !exists {
			onlyInA = append(onlyInA, s)
		}
	}
	for name, s := range bMap {
		if _, exists := aMap[name]; !exists {
			onlyInB = append(onlyInB, s)
		}
	}

	// 排序
	sort.Slice(onlyInA, func(i, j int) bool { return onlyInA[i].Name < onlyInA[j].Name })
	sort.Slice(onlyInB, func(i, j int) bool { return onlyInB[i].Name < onlyInB[j].Name })

	return &SyscallsDiff{
		OnlyInA:  onlyInA,
		OnlyInB:  onlyInB,
		TotalInA: len(a),
		TotalInB: len(b),
	}
}

// compareKernelSymbols 比较内核符号
func compareKernelSymbols(a, b []model.KernelSymbol) *KernelSymbolsDiff {
	// 构建 map 以提高查找效率
	aMap := make(map[string]model.KernelSymbol)
	bMap := make(map[string]model.KernelSymbol)

	for _, s := range a {
		aMap[s.Name] = s
	}
	for _, s := range b {
		bMap[s.Name] = s
	}

	// 查找差异
	var onlyInA, onlyInB []model.KernelSymbol
	var modified []KernelSymbolModified

	for name, symA := range aMap {
		if symB, exists := bMap[name]; exists {
			// 两者都存在，检查 CRC 是否相同
			if symA.CRC != symB.CRC {
				modified = append(modified, KernelSymbolModified{
					KernelSymbol: symA,
					CRCInA:       symA.CRC,
					CRCInB:       symB.CRC,
				})
			}
		} else {
			onlyInA = append(onlyInA, symA)
		}
	}
	for name, symB := range bMap {
		if _, exists := aMap[name]; !exists {
			onlyInB = append(onlyInB, symB)
		}
	}

	// 排序
	sort.Slice(onlyInA, func(i, j int) bool { return onlyInA[i].Name < onlyInA[j].Name })
	sort.Slice(onlyInB, func(i, j int) bool { return onlyInB[i].Name < onlyInB[j].Name })
	sort.Slice(modified, func(i, j int) bool { return modified[i].Name < modified[j].Name })

	return &KernelSymbolsDiff{
		OnlyInA:  onlyInA,
		OnlyInB:  onlyInB,
		Modified: modified,
		TotalInA: len(a),
		TotalInB: len(b),
	}
}

// compareUserspaceSymbols 比较用户态符号
func compareUserspaceSymbols(a, b []model.UserspaceSymbol) *UserspaceSymbolsDiff {
	// 按 SoPath 分组
	aByPath := groupUserspaceSymbolsByPath(a)
	bByPath := groupUserspaceSymbolsByPath(b)

	result := &UserspaceSymbolsDiff{
		BySoPath: make(map[string]*UserspaceSymbolGroupDiff),
	}

	// 收集所有 SoPath
	allPaths := make(map[string]bool)
	for path := range aByPath {
		allPaths[path] = true
	}
	for path := range bByPath {
		allPaths[path] = true
	}

	// 对每个 SoPath 进行比较
	for path := range allPaths {
		aSymbols := aByPath[path]
		bSymbols := bByPath[path]

		groupDiff := compareUserspaceSymbolGroup(aSymbols, bSymbols)
		groupDiff.SoPath = path

		result.BySoPath[path] = groupDiff

		result.TotalInA += len(aSymbols)
		result.TotalInB += len(bSymbols)
	}

	return result
}

// groupUserspaceSymbolsByPath 按 SoPath 分组用户态符号
func groupUserspaceSymbolsByPath(symbols []model.UserspaceSymbol) map[string][]model.UserspaceSymbol {
	result := make(map[string][]model.UserspaceSymbol)
	for _, s := range symbols {
		result[s.SoPath] = append(result[s.SoPath], s)
	}
	return result
}

// compareUserspaceSymbolGroup 比较单个 .so 文件的符号
func compareUserspaceSymbolGroup(a, b []model.UserspaceSymbol) *UserspaceSymbolGroupDiff {
	// 构建 map
	aMap := make(map[string]model.UserspaceSymbol)
	bMap := make(map[string]model.UserspaceSymbol)

	for _, s := range a {
		aMap[s.SymbolName] = s
	}
	for _, s := range b {
		bMap[s.SymbolName] = s
	}

	var onlyInA, onlyInB, common []model.UserspaceSymbol
	var modified []UserspaceSymbolModified

	for name, symA := range aMap {
		if symB, exists := bMap[name]; exists {
			// 两者都存在，检查版本是否相同
			if symA.SymbolVersion != symB.SymbolVersion {
				modified = append(modified, UserspaceSymbolModified{
					UserspaceSymbol: symA,
					VersionInA:      symA.SymbolVersion,
					VersionInB:      symB.SymbolVersion,
				})
			} else {
				// 版本相同
				common = append(common, symA)
			}
		} else {
			onlyInA = append(onlyInA, symA)
		}
	}
	for name, symB := range bMap {
		if _, exists := aMap[name]; !exists {
			onlyInB = append(onlyInB, symB)
		}
	}

	return &UserspaceSymbolGroupDiff{
		OnlyInA:  onlyInA,
		OnlyInB:  onlyInB,
		Modified: modified,
		Common:   common,
	}
}

// compareRPMPackages 比较 RPM 包
func compareRPMPackages(a, b []model.RPMPackage) *RPMPackagesDiff {
	// 构建 map
	aMap := make(map[string]model.RPMPackage)
	bMap := make(map[string]model.RPMPackage)

	for _, p := range a {
		aMap[p.Name] = p
	}
	for _, p := range b {
		bMap[p.Name] = p
	}

	var onlyInA, onlyInB []model.RPMPackage
	var modified []RPMPackageModified

	for name, pkgA := range aMap {
		if pkgB, exists := bMap[name]; exists {
			// 两者都存在，检查版本是否相同
			if pkgA.Version != pkgB.Version || pkgA.Release != pkgB.Release {
				// 比较版本，确定是升级还是降级
				// 如果 A 的版本 < B 的版本，说明 B 是升级
				upgrade := compareVersion(pkgA.Version, pkgB.Version) < 0
				modified = append(modified, RPMPackageModified{
					RPMPackage: pkgA,
					VersionInA: pkgA.Version + "-" + pkgA.Release,
					VersionInB: pkgB.Version + "-" + pkgB.Release,
					Upgrade:    upgrade,
				})
			}
		} else {
			onlyInA = append(onlyInA, pkgA)
		}
	}
	for name, pkgB := range bMap {
		if _, exists := aMap[name]; !exists {
			onlyInB = append(onlyInB, pkgB)
		}
	}

	// 排序
	sort.Slice(onlyInA, func(i, j int) bool { return onlyInA[i].Name < onlyInA[j].Name })
	sort.Slice(onlyInB, func(i, j int) bool { return onlyInB[i].Name < onlyInB[j].Name })
	sort.Slice(modified, func(i, j int) bool { return modified[i].Name < modified[j].Name })

	return &RPMPackagesDiff{
		OnlyInA:  onlyInA,
		OnlyInB:  onlyInB,
		Modified: modified,
		TotalInA: len(a),
		TotalInB: len(b),
	}
}

// compareVersion 比较两个版本号字符串
// 返回值: >0 表示 versionA 更新, <0 表示 versionA 更旧, 0 表示相同
func compareVersion(versionA, versionB string) int {
	// 简单的版本比较：按 . 分割成数字数组，逐个比较
	// 先去除 release 后缀（如 el7, rhel8 等）
	aCore := versionA
	bCore := versionB
	if idx := strings.Index(versionA, "-"); idx > 0 {
		aCore = versionA[:idx]
	}
	if idx := strings.Index(versionB, "-"); idx > 0 {
		bCore = versionB[:idx]
	}

	aParts := strings.Split(aCore, ".")
	bParts := strings.Split(bCore, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aNum, bNum int

		if i < len(aParts) {
			// 提取数字部分
			for _, c := range aParts[i] {
				if c >= '0' && c <= '9' {
					aNum = aNum*10 + int(c-'0')
				}
			}
		}

		if i < len(bParts) {
			for _, c := range bParts[i] {
				if c >= '0' && c <= '9' {
					bNum = bNum*10 + int(c-'0')
				}
			}
		}

		if aNum > bNum {
			return 1
		}
		if aNum < bNum {
			return -1
		}
	}

	return 0
}
