package differ

import (
	"testing"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// TestCompareVersion 测试版本比较
func TestCompareVersion(t *testing.T) {
	tests := []struct {
		name     string
		versionA string
		versionB string
		wantSign int // >0, <0, or 0
	}{
		{"equal", "2.17", "2.17", 0},
		{"A newer than B", "2.18", "2.17", 1},
		{"A older than B", "2.17", "2.18", -1},
		{"with release", "2.17.el7", "2.17.el6", 1},
		{"major version diff", "3.10", "2.17", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersion(tt.versionA, tt.versionB)
			// 简化比较，只检查符号
			wantSign := tt.wantSign
			if (got > 0 && wantSign <= 0) || (got < 0 && wantSign >= 0) || (got == 0 && wantSign != 0) {
				t.Errorf("compareVersion(%q, %q) = %d, want sign %d", tt.versionA, tt.versionB, got, wantSign)
			}
		})
	}
}

// TestCompareSyscalls 测试系统调用比较
func TestCompareSyscalls(t *testing.T) {
	osA := &model.OSSnapshot{
		Syscalls: []model.Syscall{
			{Number: 0, Name: "read"},
			{Number: 1, Name: "write"},
			{Number: 2, Name: "open"},
		},
	}

	osB := &model.OSSnapshot{
		Syscalls: []model.Syscall{
			{Number: 0, Name: "read"},
			{Number: 1, Name: "write"},
			{Number: 3, Name: "close"},
		},
	}

	result := Compare(osA, osB)

	// 验证系统调用差异
	if len(result.SyscallsDiff.OnlyInA) != 1 {
		t.Errorf("OnlyInA: expected 1, got %d", len(result.SyscallsDiff.OnlyInA))
	}
	if result.SyscallsDiff.OnlyInA[0].Name != "open" {
		t.Errorf("OnlyInA[0]: expected 'open', got %s", result.SyscallsDiff.OnlyInA[0].Name)
	}

	if len(result.SyscallsDiff.OnlyInB) != 1 {
		t.Errorf("OnlyInB: expected 1, got %d", len(result.SyscallsDiff.OnlyInB))
	}
	if result.SyscallsDiff.OnlyInB[0].Name != "close" {
		t.Errorf("OnlyInB[0]: expected 'close', got %s", result.SyscallsDiff.OnlyInB[0].Name)
	}
}

// TestCompareKernelSymbols 测试内核符号比较
func TestCompareKernelSymbols(t *testing.T) {
	osA := &model.OSSnapshot{
		KernelSymbols: []model.KernelSymbol{
			{Name: "symbol1", Module: "mod1", CRC: "0x123"},
			{Name: "symbol2", Module: "mod2", CRC: "0x456"},
			{Name: "symbol3", Module: "mod3", CRC: "0x789"},
		},
	}

	osB := &model.OSSnapshot{
		KernelSymbols: []model.KernelSymbol{
			{Name: "symbol1", Module: "mod1", CRC: "0x123"}, // 相同
			{Name: "symbol2", Module: "mod2", CRC: "0x999"}, // CRC 变化
			{Name: "symbol4", Module: "mod4", CRC: "0xabc"}, // 仅 B 有
		},
	}

	result := Compare(osA, osB)

	// 验证 Modified (CRC 变化)
	if len(result.KernelSymbolsDiff.Modified) != 1 {
		t.Errorf("Modified: expected 1, got %d", len(result.KernelSymbolsDiff.Modified))
	}
	if result.KernelSymbolsDiff.Modified[0].Name != "symbol2" {
		t.Errorf("Modified[0].Name: expected 'symbol2', got %s", result.KernelSymbolsDiff.Modified[0].Name)
	}
	if result.KernelSymbolsDiff.Modified[0].CRCInA != "0x456" || result.KernelSymbolsDiff.Modified[0].CRCInB != "0x999" {
		t.Errorf("CRC values: got %s/%s, want 0x456/0x999",
			result.KernelSymbolsDiff.Modified[0].CRCInA,
			result.KernelSymbolsDiff.Modified[0].CRCInB)
	}

	// 验证 OnlyInA
	if len(result.KernelSymbolsDiff.OnlyInA) != 1 {
		t.Errorf("OnlyInA: expected 1, got %d", len(result.KernelSymbolsDiff.OnlyInA))
	}
	if result.KernelSymbolsDiff.OnlyInA[0].Name != "symbol3" {
		t.Errorf("OnlyInA[0]: expected 'symbol3', got %s", result.KernelSymbolsDiff.OnlyInA[0].Name)
	}

	// 验证 OnlyInB
	if len(result.KernelSymbolsDiff.OnlyInB) != 1 {
		t.Errorf("OnlyInB: expected 1, got %d", len(result.KernelSymbolsDiff.OnlyInB))
	}
	if result.KernelSymbolsDiff.OnlyInB[0].Name != "symbol4" {
		t.Errorf("OnlyInB[0]: expected 'symbol4', got %s", result.KernelSymbolsDiff.OnlyInB[0].Name)
	}
}

// TestCompareUserspaceSymbols 测试用户态符号比较
func TestCompareUserspaceSymbols(t *testing.T) {
	osA := &model.OSSnapshot{
		UserspaceSymbols: []model.UserspaceSymbol{
			{SoPath: "/lib64/libc.so.6", SymbolName: "malloc", SymbolVersion: "GLIBC_2.17"},
			{SoPath: "/lib64/libc.so.6", SymbolName: "free", SymbolVersion: "GLIBC_2.17"},
			{SoPath: "/lib64/libc.so.6", SymbolName: "pthread_create", SymbolVersion: "GLIBC_2.34"},
		},
	}

	osB := &model.OSSnapshot{
		UserspaceSymbols: []model.UserspaceSymbol{
			{SoPath: "/lib64/libc.so.6", SymbolName: "malloc", SymbolVersion: "GLIBC_2.17"},
			{SoPath: "/lib64/libc.so.6", SymbolName: "free", SymbolVersion: "GLIBC_2.17"},
			{SoPath: "/lib64/libc.so.6", SymbolName: "pthread_create", SymbolVersion: "GLIBC_2.17"}, // 降级
			{SoPath: "/lib64/libc.so.6", SymbolName: "calloc", SymbolVersion: "GLIBC_2.17"},         // 新增
		},
	}

	result := Compare(osA, osB)

	path := "/lib64/libc.so.6"
	groupDiff := result.UserspaceSymbolsDiff.BySoPath[path]

	// 验证 Modified (版本变化)
	if len(groupDiff.Modified) != 1 {
		t.Errorf("Modified: expected 1, got %d", len(groupDiff.Modified))
	}
	if groupDiff.Modified[0].SymbolName != "pthread_create" {
		t.Errorf("Modified[0].SymbolName: expected 'pthread_create', got %s", groupDiff.Modified[0].SymbolName)
	}

	// 验证版本降级
	if groupDiff.Modified[0].VersionInA != "GLIBC_2.34" || groupDiff.Modified[0].VersionInB != "GLIBC_2.17" {
		t.Errorf("Version values: got %s/%s, want GLIBC_2.34/GLIBC_2.17",
			groupDiff.Modified[0].VersionInA,
			groupDiff.Modified[0].VersionInB)
	}

	// 验证 OnlyInB (新增)
	if len(groupDiff.OnlyInB) != 1 {
		t.Errorf("OnlyInB: expected 1, got %d", len(groupDiff.OnlyInB))
	}
	if groupDiff.OnlyInB[0].SymbolName != "calloc" {
		t.Errorf("OnlyInB[0].SymbolName: expected 'calloc', got %s", groupDiff.OnlyInB[0].SymbolName)
	}
}

// TestCompareRPMPackages 测试 RPM 包比较
func TestCompareRPMPackages(t *testing.T) {
	osA := &model.OSSnapshot{
		RPMPackages: []model.RPMPackage{
			{Name: "glibc", Version: "2.17", Release: "el7", Arch: "x86_64"},
			{Name: "openssl", Version: "1.0.2", Release: "el7", Arch: "x86_64"},
			{Name: "bash", Version: "4.2", Release: "el7", Arch: "x86_64"},
		},
	}

	osB := &model.OSSnapshot{
		RPMPackages: []model.RPMPackage{
			{Name: "glibc", Version: "2.17", Release: "el7", Arch: "x86_64"},    // 相同
			{Name: "openssl", Version: "1.1.1", Release: "el7", Arch: "x86_64"}, // 升级
			{Name: "curl", Version: "7.61", Release: "el7", Arch: "x86_64"},     // 新增
		},
	}

	result := Compare(osA, osB)

	// 验证 Modified
	if len(result.RPMPackagesDiff.Modified) != 1 {
		t.Errorf("Modified: expected 1, got %d", len(result.RPMPackagesDiff.Modified))
	}
	if result.RPMPackagesDiff.Modified[0].Name != "openssl" {
		t.Errorf("Modified[0].Name: expected 'openssl', got %s", result.RPMPackagesDiff.Modified[0].Name)
	}

	// 验证是升级
	if !result.RPMPackagesDiff.Modified[0].Upgrade {
		t.Error("Expected Upgrade to be true")
	}

	// 验证 OnlyInA (删除)
	if len(result.RPMPackagesDiff.OnlyInA) != 1 {
		t.Errorf("OnlyInA: expected 1, got %d", len(result.RPMPackagesDiff.OnlyInA))
	}
	if result.RPMPackagesDiff.OnlyInA[0].Name != "bash" {
		t.Errorf("OnlyInA[0].Name: expected 'bash', got %s", result.RPMPackagesDiff.OnlyInA[0].Name)
	}

	// 验证 OnlyInB (新增)
	if len(result.RPMPackagesDiff.OnlyInB) != 1 {
		t.Errorf("OnlyInB: expected 1, got %d", len(result.RPMPackagesDiff.OnlyInB))
	}
	if result.RPMPackagesDiff.OnlyInB[0].Name != "curl" {
		t.Errorf("OnlyInB[0].Name: expected 'curl', got %s", result.RPMPackagesDiff.OnlyInB[0].Name)
	}
}
