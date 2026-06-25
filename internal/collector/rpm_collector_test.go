package collector

import (
	"testing"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// TestCollectRPMPackages_Mock 测试 CollectRPMPackages 函数，使用 mock 命令输出。
// 该测试不需要 root 权限，也不依赖真实的 rpm 命令。
func TestCollectRPMPackages_Mock(t *testing.T) {
	// Mock rpm 命令输出
	mockOutput := `glibc 2.17 el7 x86_64
openssl 1.0.2k el7 x86_64
bash 4.2.46 el7 x86_64
kernel 3.10.0 el7 x86_64
invalid-line
coreutils 8.22 el7 x86_64
`
	// 使用自定义执行器来返回 mock 输出
	executor := func() (string, error) {
		return mockOutput, nil
	}

	// 执行测试
	packages, err := collectRPMPackagesWithExecutor(executor)
	if err != nil {
		t.Fatalf("collectRPMPackagesWithExecutor() returned unexpected error: %v", err)
	}

	// 验证结果数量（跳过空行和无效行：5个有效 + 1个无效行 = 6行，invalid-line 被跳过）
	expectedCount := 5
	if len(packages) != expectedCount {
		t.Errorf("expected %d packages, got %d", expectedCount, len(packages))
	}

	// 验证第一个包
	if len(packages) > 0 {
		expectedPkg := model.RPMPackage{
			Name:    "glibc",
			Version: "2.17",
			Release: "el7",
			Arch:    "x86_64",
		}
		if packages[0] != expectedPkg {
			t.Errorf("first package mismatch:\nexpected: %+v\ngot: %+v", expectedPkg, packages[0])
		}
	}

	// 验证跳过了无效行
	if len(packages) != 5 {
		t.Errorf("invalid line should be skipped, expected 5 packages, got %d", len(packages))
	}

	// 验证最后一个包是 coreutils
	if len(packages) == 5 {
		expectedLastPkg := model.RPMPackage{
			Name:    "coreutils",
			Version: "8.22",
			Release: "el7",
			Arch:    "x86_64",
		}
		if packages[4] != expectedLastPkg {
			t.Errorf("last package mismatch:\nexpected: %+v\ngot: %+v", expectedLastPkg, packages[4])
		}
	}
}

// TestParseRPMPackages 测试 parseRPMPackages 函数的解析逻辑。
func TestParseRPMPackages(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantCount int
		wantFirst model.RPMPackage
		wantErr   bool
	}{
		{
			name:      "normal output",
			output:    "glibc 2.17 el7 x86_64\nopenssl 1.0.2k el7 x86_64\n",
			wantCount: 2,
			wantFirst: model.RPMPackage{
				Name: "glibc", Version: "2.17", Release: "el7", Arch: "x86_64",
			},
		},
		{
			name:      "empty output",
			output:    "",
			wantCount: 0,
		},
		{
			name:      "output with only empty lines",
			output:    "\n\n\n",
			wantCount: 0,
		},
		{
			name:      "malformed line skipped",
			output:    "glibc 2.17 el7 x86_64\ninvalid\nopenssl 1.0.2k el7 x86_64\n",
			wantCount: 2,
			wantFirst: model.RPMPackage{
				Name: "glibc", Version: "2.17", Release: "el7", Arch: "x86_64",
			},
		},
		{
			name:      "multiple spaces between fields",
			output:    "pkg1  1.0  el7  x86_64\n",
			wantCount: 1,
			wantFirst: model.RPMPackage{
				Name: "pkg1", Version: "1.0", Release: "el7", Arch: "x86_64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages, err := parseRPMPackages(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRPMPackages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(packages) != tt.wantCount {
				t.Errorf("parseRPMPackages() got %d packages, want %d", len(packages), tt.wantCount)
			}
			if tt.wantCount > 0 && packages[0] != tt.wantFirst {
				t.Errorf("first package = %+v, want %+v", packages[0], tt.wantFirst)
			}
		})
	}
}

// TestCollectRPMPackages_RealCommand 测试 CollectRPMPackages 函数的集成测试。
// 该测试需要 root 权限且系统安装了 rpm 命令。
//
// 在真实环境下运行方式（需要 root 权限）：
//
//	go test -v -run TestCollectRPMPackages_RealCommand ./internal/collector/
func TestCollectRPMPackages_RealCommand(t *testing.T) {
	t.Skip("skipping real command test, run with -tags integration to execute")

	packages, err := CollectRPMPackages()
	if err != nil {
		t.Fatalf("CollectRPMPackages() returned unexpected error: %v", err)
	}

	if len(packages) == 0 {
		t.Error("expected at least one package, got none")
	}

	// 验证返回的包结构体字段非空
	for _, pkg := range packages {
		if pkg.Name == "" {
			t.Error("package name should not be empty")
		}
	}
}
