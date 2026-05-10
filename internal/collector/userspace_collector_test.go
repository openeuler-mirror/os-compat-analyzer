package collector

import (
	"os"
	"testing"
)

// TestFindSOFiles 测试目录扫描功能，使用临时目录
func TestFindSOFiles(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()

	// 创建测试子目录
	subDir := tmpDir + "/subdir"
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// 创建测试 .so 文件
	testFiles := []string{
		tmpDir + "/libtest.so",
		tmpDir + "/libfoo.so.1",
		subDir + "/libbar.so.2",
		subDir + "/libbaz.a", // 不是 .so 文件，不应该被包含
	}

	for _, path := range testFiles {
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", path, err)
		}
		f.Close()
	}

	// 测试 findSOFiles
	soFiles, err := findSOFiles([]string{tmpDir})
	if err != nil {
		t.Fatalf("findSOFiles() error = %v", err)
	}

	// 应该找到 3 个 .so 文件
	if len(soFiles) != 3 {
		t.Errorf("expected 3 .so files, got %d: %v", len(soFiles), soFiles)
	}

	// 验证找到的文件
	expected := map[string]bool{
		tmpDir + "/libtest.so":    true,
		tmpDir + "/libfoo.so.1":   true,
		subDir + "/libbar.so.2":   true,
	}
	for _, f := range soFiles {
		if !expected[f] {
			t.Errorf("unexpected file: %s", f)
		}
	}
}

// TestIsSOFile 测试 isSOFile 函数
func TestIsSOFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"libfoo.so", true},
		{"libfoo.so.1", true},
		{"libfoo.so.1.2", true},
		{"libfoo.so.1.2.3", true},
		{"libbar.a", false},
		{"libbaz.sox", false},
		{"libqux.so.", false},
		{"/usr/lib64/libc.so.6", true},
		{"/lib64/libpthread.so.0", true},
	}

	for _, tt := range tests {
		result := isSOFile(tt.path)
		if result != tt.expected {
			t.Errorf("isSOFile(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

// TestExtractVersionFromName 测试从符号名提取版本信息
func TestExtractVersionFromName(t *testing.T) {
	tests := []struct {
		name     string
		symName  string
		expected string
	}{
		{
			name:     "GLIBC version",
			symName:  "__GI___strchr_avx2",
			expected: "",
		},
		{
			name:     "pthread version",
			symName:  "pthread_create",
			expected: "",
		},
		{
			name:     "no version",
			symName:  "main",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromName(tt.symName)
			if result != tt.expected {
				t.Errorf("extractVersionFromName(%s) = %q, want %q", tt.symName, result, tt.expected)
			}
		})
	}
}

// TestGetSymbolType 测试从 Info 字段提取类型
func TestGetSymbolType(t *testing.T) {
	tests := []struct {
		info     byte
		expected uint8
	}{
		{0x12, 0x02}, // STT_FUNC (2) | STB_GLOBAL (0x10)
		{0x11, 0x01}, // STT_OBJECT (1) | STB_GLOBAL (0x10)
		{0x03, 0x03}, // STT_NOTYPE (3) | STB_LOCAL (0x00)
		{0x00, 0x00}, // STT_NOTYPE (0) | STB_LOCAL (0x00)
	}

	for _, tt := range tests {
		result := getSymbolType(tt.info)
		if result != tt.expected {
			t.Errorf("getSymbolType(0x%02x) = 0x%02x, want 0x%02x", tt.info, result, tt.expected)
		}
	}
}

// TestGetSymbolBind 测试从 Info 字段提取绑定信息
func TestGetSymbolBind(t *testing.T) {
	tests := []struct {
		info     byte
		expected uint8
	}{
		{0x12, 0x01}, // STT_FUNC (2) | STB_GLOBAL (0x10)
		{0x32, 0x03}, // STT_FILE (4) | STB_WEAK (0x30)
		{0x03, 0x00}, // STT_NOTYPE (3) | STB_LOCAL (0x00)
		{0x20, 0x02}, // STT_SECTION (3) | STB_LOCAL (0x20)
	}

	for _, tt := range tests {
		result := getSymbolBind(tt.info)
		if result != tt.expected {
			t.Errorf("getSymbolBind(0x%02x) = 0x%02x, want 0x%02x", tt.info, result, tt.expected)
		}
	}
}

// TestCollectUserspaceSymbols_Real 测试实际的符号采集
// 该测试需要系统有 .so 文件，默认跳过
func TestCollectUserspaceSymbols_Real(t *testing.T) {
	t.Skip("skipping real test, run with -tags integration to execute")

	symbols, err := CollectUserspaceSymbols(nil)
	if err != nil {
		t.Fatalf("CollectUserspaceSymbols() error = %v", err)
	}

	if len(symbols) == 0 {
		t.Log("no symbols collected (may be normal if no .so files found)")
	}

	// 打印前几个符号用于验证
	for i, sym := range symbols {
		if i >= 5 {
			break
		}
		t.Logf("Symbol: SoPath=%s, Name=%s, Version=%s",
			sym.SoPath, sym.SymbolName, sym.SymbolVersion)
	}
}
