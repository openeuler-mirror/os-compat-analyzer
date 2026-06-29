package collector

import (
	"testing"
)

// TestFindKey 测试 findKey 函数
func TestFindKey(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		key      string
		expected int
	}{
		{
			name:     "found at beginning",
			text:     "PRETTY_NAME=\"openEuler\"",
			key:      "PRETTY_NAME",
			expected: 0,
		},
		{
			name:     "found in middle",
			text:     "NAME=\"Linux\"\nPRETTY_NAME=\"openEuler\"",
			key:      "PRETTY_NAME",
			expected: 13,
		},
		{
			name:     "not found",
			text:     "NAME=\"Linux\"",
			key:      "PRETTY_NAME",
			expected: -1,
		},
		{
			name:     "empty text",
			text:     "",
			key:      "PRETTY_NAME",
			expected: -1,
		},
		{
			name:     "empty key",
			text:     "PRETTY_NAME=\"openEuler\"",
			key:      "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findKey(tt.text, tt.key)
			if got != tt.expected {
				t.Errorf("findKey(%q, %q) = %d, want %d", tt.text, tt.key, got, tt.expected)
			}
		})
	}
}

// TestExtractValue 测试 extractValue 函数
func TestExtractValue(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		key      string
		expected string
	}{
		{
			name:     "quoted value",
			text:     "PRETTY_NAME=\"openEuler 20.03\"",
			key:      "PRETTY_NAME",
			expected: "openEuler 20.03",
		},
		{
			name:     "unquoted value",
			text:     "PRETTY_NAME=openEuler",
			key:      "PRETTY_NAME",
			expected: "openEuler",
		},
		{
			name:     "no equals sign",
			text:     "PRETTY_NAME",
			key:      "PRETTY_NAME",
			expected: "",
		},
		{
			name:     "empty value",
			text:     "PRETTY_NAME=",
			key:      "PRETTY_NAME",
			expected: "",
		},
		{
			name:     "value with special characters",
			text:     "PRETTY_NAME=\"openEuler (LTS)\"",
			key:      "PRETTY_NAME",
			expected: "openEuler (LTS)",
		},
		{
			name:     "next line has equals sign",
			text:     "PRETTY_NAME=\"openEuler 24.03 (LTS-SP3)\"\nANSI_COLOR=\"0;31\"",
			key:      "PRETTY_NAME",
			expected: "openEuler 24.03 (LTS-SP3)",
		},
		{
			name:     "mismatch key",
			text:     "NAME=\"Linux\"",
			key:      "PRETTY_NAME",
			expected: "",
		},
		{
			name:     "empty key",
			text:     "PRETTY_NAME=\"openEuler\"",
			key:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractValue(tt.text, tt.key)
			if got != tt.expected {
				t.Errorf("extractValue(%q, %q) = %q, want %q", tt.text, tt.key, got, tt.expected)
			}
		})
	}
}

// TestCollectOSMetadataWithInputs 测试 collectOSMetadataWithInputs 函数
func TestCollectOSMetadataWithInputs(t *testing.T) {
	tests := []struct {
		name                 string
		osRelease            string
		kernelVersion        string
		architecture         string
		currentUser          string
		expectedName         string
		expectedVersion      string
		expectedArchitecture string
		expectedUser         string
	}{
		{
			name: "all fields present",
			osRelease: `NAME="openEuler"
VERSION_ID="20.03"
PRETTY_NAME="openEuler 20.03 (LTS)"
`,
			kernelVersion:        "4.19.90-2107.6.0.0100.oe1.x86_64\n",
			architecture:         "x86_64\n",
			currentUser:          "root",
			expectedName:         "openEuler 20.03 (LTS)",
			expectedVersion:      "4.19.90-2107.6.0.0100.oe1.x86_64",
			expectedArchitecture: "x86_64",
			expectedUser:         "root",
		},
		{
			name: "missing pretty name",
			osRelease: `NAME="openEuler"
VERSION_ID="20.03"
`,
			kernelVersion:        "5.10.0\n",
			architecture:         "aarch64\n",
			currentUser:          "admin",
			expectedName:         "",
			expectedVersion:      "5.10.0",
			expectedArchitecture: "aarch64",
			expectedUser:         "admin",
		},
		{
			name:                 "empty inputs",
			osRelease:            "",
			kernelVersion:        "",
			architecture:         "",
			currentUser:          "",
			expectedName:         "",
			expectedVersion:      "",
			expectedArchitecture: "",
			expectedUser:         "",
		},
		{
			name: "whitespace trimming",
			osRelease: `PRETTY_NAME="openEuler 20.03"
`,
			kernelVersion:        "  5.10.0-oe1  \n",
			architecture:         "  x86_64  \n",
			currentUser:          "  testuser  \n",
			expectedName:         "openEuler 20.03",
			expectedVersion:      "5.10.0-oe1",
			expectedArchitecture: "x86_64",
			expectedUser:         "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectOSMetadataWithInputs(tt.osRelease, tt.kernelVersion, tt.architecture, tt.currentUser)

			if got.Name != tt.expectedName {
				t.Errorf("Name = %q, want %q", got.Name, tt.expectedName)
			}
			if got.Version != tt.expectedVersion {
				t.Errorf("Version = %q, want %q", got.Version, tt.expectedVersion)
			}
			if got.Architecture != tt.expectedArchitecture {
				t.Errorf("Architecture = %q, want %q", got.Architecture, tt.expectedArchitecture)
			}
			if got.User != tt.expectedUser {
				t.Errorf("User = %q, want %q", got.User, tt.expectedUser)
			}
			if got.CollectedAt.IsZero() {
				t.Error("CollectedAt should not be zero")
			}
		})
	}
}

// TestCollectOSMetadataWithInputs_NameExtraction 验证 PRETTY_NAME 提取逻辑
func TestCollectOSMetadataWithInputs_NameExtraction(t *testing.T) {
	tests := []struct {
		name         string
		osRelease    string
		expectedName string
	}{
		{
			name: "first occurrence used",
			osRelease: `PRETTY_NAME="First"
PRETTY_NAME="Second"
`,
			expectedName: "First",
		},
		{
			name: "substring match not at boundary",
			osRelease: `PRETTY_NAME_FULL="Should Not Match"
PRETTY_NAME="Correct"
`,
			expectedName: "Correct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectOSMetadataWithInputs(tt.osRelease, "", "", "")
			if got.Name != tt.expectedName {
				t.Errorf("Name = %q, want %q", got.Name, tt.expectedName)
			}
		})
	}
}

// TestCollectOSMetadata_Real 测试真实的 OS 元数据采集
// 该测试需要读取真实的系统文件，默认跳过
func TestCollectOSMetadata_Real(t *testing.T) {
	t.Skip("skipping real test, run with -tags integration to execute")

	metadata := CollectOSMetadata()

	if metadata.Name == "" {
		t.Error("expected non-empty OS name")
	}
	if metadata.Version == "" {
		t.Error("expected non-empty kernel version")
	}
	if metadata.Architecture == "" {
		t.Error("expected non-empty architecture")
	}
	if metadata.CollectedAt.IsZero() {
		t.Error("expected non-zero collected at time")
	}

	t.Logf("OS Metadata: Name=%s, Version=%s, Architecture=%s, CollectedAt=%v",
		metadata.Name, metadata.Version, metadata.Architecture, metadata.CollectedAt)
}
