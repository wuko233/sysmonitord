package pathmatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// 测试 HasGlobMeta
func TestHasGlobMeta(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"普通路径", "/home/user/file.txt", false},
		{"包含星号", "/home/*/file.txt", true},
		{"包含问号", "/home/user/file?.txt", true},
		{"包含方括号", "/home/[abc]/file.txt", true},
		{"包含花括号", "/home/{a,b}/file.txt", true},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasGlobMeta(tt.path)
			if result != tt.expected {
				t.Errorf("HasGlobMeta(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// 测试 ExpandPaths
func TestExpandPaths(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir := t.TempDir()

	// 创建测试文件结构
	files := []string{
		"file1.txt",
		"file2.txt",
		"test.log",
		"data/file3.txt",
		"data/file4.log",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		patterns []string
		expected []string // 注意：实际结果依赖于文件系统
	}{
		{
			name:     "普通路径",
			patterns: []string{filepath.Join(tmpDir, "file1.txt")},
			expected: []string{filepath.Join(tmpDir, "file1.txt")},
		},
		{
			name:     "通配符匹配多个文件",
			patterns: []string{filepath.Join(tmpDir, "*.txt")},
			expected: []string{
				filepath.Join(tmpDir, "file1.txt"),
				filepath.Join(tmpDir, "file2.txt"),
			},
		},
		{
			name: "多个模式",
			patterns: []string{
				filepath.Join(tmpDir, "*.txt"),
				filepath.Join(tmpDir, "*.log"),
			},
			expected: []string{
				filepath.Join(tmpDir, "file1.txt"),
				filepath.Join(tmpDir, "file2.txt"),
				filepath.Join(tmpDir, "test.log"),
			},
		},
		{
			name:     "嵌套目录",
			patterns: []string{filepath.Join(tmpDir, "data", "*.txt")},
			expected: []string{filepath.Join(tmpDir, "data", "file3.txt")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPaths(tt.patterns)
			if err != nil {
				t.Fatalf("ExpandPaths returned error: %v", err)
			}

			// 比较结果（顺序无关）
			if !compareSlices(result, tt.expected) {
				t.Errorf("ExpandPaths(%v) = %v, want %v", tt.patterns, result, tt.expected)
			}
		})
	}
}

// 测试 MatchPath
func TestMatchPath(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{
			name:     "精确匹配",
			pattern:  "/home/user/file.txt",
			path:     "/home/user/file.txt",
			expected: true,
		},
		{
			name:     "不匹配",
			pattern:  "/home/user/file.txt",
			path:     "/home/user/other.txt",
			expected: false,
		},
		{
			name:     "前缀匹配 - 目录",
			pattern:  "/home/user",
			path:     "/home/user/file.txt",
			expected: true,
		},
		{
			name:     "前缀匹配 - 子目录",
			pattern:  "/home/user",
			path:     "/home/user/sub/file.txt",
			expected: true,
		},
		{
			name:     "前缀不匹配",
			pattern:  "/home/user",
			path:     "/home/other/file.txt",
			expected: false,
		},
		{
			name:     "通配符匹配",
			pattern:  "/home/*/file.txt",
			path:     "/home/user/file.txt",
			expected: true,
		},
		{
			name:     "通配符不匹配",
			pattern:  "/home/*/file.txt",
			path:     "/home/user/sub/file.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchPath(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("MatchPath(%q, %q) = %v, want %v",
					tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

// 测试 isPathExcluded（通过导出函数测试私有函数）
func TestIsPathExcluded(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		excludes []string
		expected bool
	}{
		{
			name:     "精确排除",
			path:     "/home/user/file.txt",
			excludes: []string{"/home/user/file.txt"},
			expected: true,
		},
		{
			name:     "目录排除",
			path:     "/home/user/file.txt",
			excludes: []string{"/home/user"},
			expected: true,
		},
		{
			name:     "通配符排除",
			path:     "/home/user/test.log",
			excludes: []string{"/home/*/*.log"},
			expected: true,
		},
		{
			name:     "未被排除",
			path:     "/home/user/file.txt",
			excludes: []string{"/tmp/**"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println("测试路径:", tt.path, "排除路径:", tt.excludes)
			result := IsPathExcluded(tt.path, tt.excludes)
			fmt.Println("结果:", result, "期望:", tt.expected)
			if result != tt.expected {
				t.Errorf("isPathExcluded(%q, %v) = %v, want %v",
					tt.path, tt.excludes, result, tt.expected)
			}
		})
	}
}

// 辅助函数：比较两个切片是否包含相同元素（顺序无关）
func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	ma := make(map[string]int)
	for _, s := range a {
		ma[s]++
	}
	for _, s := range b {
		ma[s]--
		if ma[s] < 0 {
			return false
		}
	}
	return true
}

// 基准测试
func BenchmarkExpandPaths(b *testing.B) {
	tmpDir := b.TempDir()
	patterns := []string{filepath.Join(tmpDir, "*.txt")}

	// 创建一些文件
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(path, []byte("test"), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExpandPaths(patterns)
	}
}

func BenchmarkMatchPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MatchPath("/home/*/file.txt", "/home/user/file.txt")
	}
}
