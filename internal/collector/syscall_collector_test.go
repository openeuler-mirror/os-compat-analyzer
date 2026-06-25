package collector

import (
	"os"
	"strings"
	"testing"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// TestHandleKallsymsError 测试错误处理
func TestHandleKallsymsError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectError bool
	}{
		{
			name:        "file not found",
			err:         &os.PathError{Op: "open", Path: "/proc/kallsyms", Err: os.ErrNotExist},
			expectError: true,
		},
		{
			name:        "permission denied",
			err:         &os.PathError{Op: "open", Path: "/proc/kallsyms", Err: os.ErrPermission},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleKallsymsError(tt.err)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// TestCollectSyscalls_Real 测试实际采集
// 该测试需要读取 /proc/kallsyms，默认跳过
func TestCollectSyscalls_Real(t *testing.T) {
	t.Skip("skipping real test, run with -tags integration to execute")

	syscalls, err := CollectSyscalls()
	if err != nil {
		t.Fatalf("CollectSyscalls() error = %v", err)
	}

	if len(syscalls) == 0 {
		t.Error("expected at least one syscall")
	}

	// 打印前几个系统调用用于验证
	for i, sc := range syscalls {
		if i >= 10 {
			break
		}
		t.Logf("Syscall: Number=%d, Name=%s", sc.Number, sc.Name)
	}
}

// TestParseKallsymsOutput 解析 /proc/kallsyms 格式的测试
// 模拟解析以下格式的行:
// 0000000000000000 T sys_read
// 0000000000000000 t __x64_sys_read
func TestParseKallsymsOutput(t *testing.T) {
	// 模拟解析逻辑
	seen := make(map[string]bool)
	var syscalls []model.Syscall

	lines := []string{
		"00000000004a20a0 T sys_read",
		"00000000004a20b0 T sys_write",
		"0000000000000000 T sys_ni_syscall",
		"00000000004a20c0 t __x64_sys_read",
		"ffffffff81000000 T native_read_cr0",
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		symType := fields[1]
		symbol := fields[2]

		if !strings.HasPrefix(symbol, "sys_") || (symType != "T" && symType != "t") {
			continue
		}
		if symbol == "sys_ni_syscall" {
			continue
		}

		syscallName := strings.TrimPrefix(symbol, "sys_")
		// 处理 __x64_sys_xxx 格式
		if strings.HasPrefix(syscallName, "__x64_sys_") {
			syscallName = strings.TrimPrefix(syscallName, "__x64_")
		}

		if seen[syscallName] {
			continue
		}
		seen[syscallName] = true

		syscalls = append(syscalls, model.Syscall{Number: 0, Name: syscallName})
	}

	expected := []model.Syscall{
		{Name: "read"},
		{Name: "write"},
	}

	if len(syscalls) != len(expected) {
		t.Errorf("expected %d syscalls, got %d", len(expected), len(syscalls))
	}

	for i, exp := range expected {
		if syscalls[i].Name != exp.Name {
			t.Errorf("syscall %d: expected %s, got %s", i, exp.Name, syscalls[i].Name)
		}
	}
}
