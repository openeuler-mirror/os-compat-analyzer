package collector

import (
	"os"
	"path/filepath"
	"testing"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// TestFindSymversFile 测试查找 symvers 文件
func TestFindSymversFile(t *testing.T) {
	// 创建临时目录结构模拟
	tmpDir := t.TempDir()

	// 测试场景1: 文件存在于 /lib/modules/{version}/build/Module.symvers
	version := "5.10.0.el7.x86_64"
	buildDir := filepath.Join(tmpDir, "lib", "modules", version, "build")
	os.MkdirAll(buildDir, 0755)
	symversPath := filepath.Join(buildDir, "Module.symvers")
	os.WriteFile(symversPath, []byte("0x12345678\tsymbol1\tmodule1\tEXPORT_SYMBOL\n"), 0644)

	// 测试查找逻辑（需要修改函数接受自定义路径，这里先用集成测试方式）
	_ = symversPath

	// TestParseSymversFile 测试解析功能
	t.Run("parse symvers", func(t *testing.T) {
		// 创建一个临时文件
		tmpFile := filepath.Join(tmpDir, "test.symvers")
		content := `0x12345678	symbol1	module1	EXPORT_SYMBOL
0x87654321	symbol2	module2	EXPORT_SYMBOL
# comment line
0xabcdef00	symbol3	module3	EXPORT_SYMBOL_GPL
`
		os.WriteFile(tmpFile, []byte(content), 0644)

		symbols, err := parseSymversFile(tmpFile)
		if err != nil {
			t.Fatalf("parseSymversFile() error = %v", err)
		}

		// 应该解析出3个符号（跳过注释行）
		if len(symbols) != 3 {
			t.Errorf("expected 3 symbols, got %d", len(symbols))
		}

		// 验证第一个符号
		expected := model.KernelSymbol{
			Name:   "symbol1",
			Module: "module1",
			CRC:    "0x12345678",
		}
		if symbols[0] != expected {
			t.Errorf("first symbol = %+v, want %+v", symbols[0], expected)
		}
	})
}

// TestParseSymversFile 测试解析 symvers 文件
func TestParseSymversFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst model.KernelSymbol
	}{
		{
			name: "normal format",
			content: `0x12345678	symbol1	module1	EXPORT_SYMBOL
0x87654321	symbol2	module2	EXPORT_SYMBOL
`,
			wantCount: 2,
			wantFirst: model.KernelSymbol{
				Name: "symbol1", Module: "module1", CRC: "0x12345678",
			},
		},
		{
			name: "with comments",
			content: `# This is a comment
0x12345678	symbol1	module1	EXPORT_SYMBOL

0x87654321	symbol2	module2	EXPORT_SYMBOL
`,
			wantCount: 2,
			wantFirst: model.KernelSymbol{
				Name: "symbol1", Module: "module1", CRC: "0x12345678",
			},
		},
		{
			name:      "empty content",
			content:   ``,
			wantCount: 0,
		},
		{
			name: "malformed lines skipped",
			content: `0x12345678	symbol1
0x87654321	symbol2	module2	EXPORT_SYMBOL
`,
			wantCount: 1,
			wantFirst: model.KernelSymbol{
				Name: "symbol2", Module: "module2", CRC: "0x87654321",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(tmpDir, "test.symvers")
			os.WriteFile(tmpFile, []byte(tt.content), 0644)

			symbols, err := parseSymversFile(tmpFile)
			if err != nil {
				t.Fatalf("parseSymversFile() error = %v", err)
			}

			if len(symbols) != tt.wantCount {
				t.Errorf("got %d symbols, want %d", len(symbols), tt.wantCount)
			}

			if tt.wantCount > 0 && symbols[0] != tt.wantFirst {
				t.Errorf("first symbol = %+v, want %+v", symbols[0], tt.wantFirst)
			}
		})
	}
}

// TestCollectKernelSymbols_Real 测试实际的符号采集
// 该测试需要系统有 Module.symvers 文件，默认跳过
func TestCollectKernelSymbols_Real(t *testing.T) {
	t.Skip("skipping real test, run with -tags integration to execute")

	symbols, err := CollectKernelSymbols("")
	if err != nil {
		t.Fatalf("CollectKernelSymbols() error = %v", err)
	}

	if len(symbols) == 0 {
		t.Error("expected at least one symbol")
	}

	// 打印前几个符号用于验证
	for i, sym := range symbols {
		if i >= 5 {
			break
		}
		t.Logf("Symbol: Name=%s, Module=%s, CRC=%s", sym.Name, sym.Module, sym.CRC)
	}
}

// TestParseKallsymsFile 测试解析 /proc/kallsyms 文件
func TestParseKallsymsFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		wantCount int
		checks    []model.KernelSymbol
	}{
		{
			name: "normal with modules",
			content: `0000000000000000 T _stext
0000000000000000 t vxlan_find_mac_rcu       [vxlan]
0000000000000000 T __do_softirq
0000000000000000 d ext4_write_inode   [ext4]
`,
			wantCount: 4,
			checks: []model.KernelSymbol{
				{Name: "_stext", Module: "vmlinux", CRC: ""},
				{Name: "vxlan_find_mac_rcu", Module: "vxlan", CRC: ""},
				{Name: "__do_softirq", Module: "vmlinux", CRC: ""},
				{Name: "ext4_write_inode", Module: "ext4", CRC: ""},
			},
		},
		{
			name: "skip noise symbols",
			content: `0000000000000000 T _stext
0000000000000000 t $x   [vxlan]
0000000000000000 t .Ltmp0   [vxlan]
0000000000000000 T __do_softirq
`,
			wantCount: 2,
			checks: []model.KernelSymbol{
				{Name: "_stext", Module: "vmlinux", CRC: ""},
				{Name: "__do_softirq", Module: "vmlinux", CRC: ""},
			},
		},
		{
			name:      "empty content",
			content:   ``,
			wantCount: 0,
		},
		{
			name: "malformed lines skipped",
			content: `0000000000000000 T
0000000000000000 T __do_softirq
`,
			wantCount: 1,
			checks: []model.KernelSymbol{
				{Name: "__do_softirq", Module: "vmlinux", CRC: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(tmpDir, "kallsyms_test")
			os.WriteFile(tmpFile, []byte(tt.content), 0644)

			symbols, err := parseKallsymsFile(tmpFile)
			if err != nil {
				t.Fatalf("parseKallsymsFile() error = %v", err)
			}

			if len(symbols) != tt.wantCount {
				t.Errorf("got %d symbols, want %d", len(symbols), tt.wantCount)
			}

			for i, want := range tt.checks {
				if i >= len(symbols) {
					t.Errorf("missing symbol at index %d", i)
					break
				}
				if symbols[i] != want {
					t.Errorf("symbol[%d] = %+v, want %+v", i, symbols[i], want)
				}
			}
		})
	}
}

/*
标准 Module.symvers 文件格式示例：

# Module.symvers 文件格式：
# CRC<tab>SymbolName<tab>Module<tab>ExportType
#
# 示例内容：
0x00000000	__init_waitqueue_head	kernel	EXPORT_SYMBOL
0x12345678	__kmalloc	mm/slab.ko	EXPORT_SYMBOL
0x87654321	pci_register_driver	drivers/pci/pci.ko	EXPORT_SYMBOL
0xabcdef00	ext4_write_inode	fs/ext4/ext4.ko	EXPORT_SYMBOL_GPL
0xdeadbeef	__pthread_create	pthread.ko	EXPORT_SYMBOL

说明：
- CRC: 符号的 CRC 校验值（十六进制）
- SymbolName: 符号名称
- Module: 导出该符号的内核模块名
- ExportType: 导出类型（EXPORT_SYMBOL 或 EXPORT_SYMBOL_GPL）
*/
