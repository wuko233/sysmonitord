package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"sysmonitord/internal/config"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
)

func TestGetRootFromPattern(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.MkdirAll(tmpDir+"/testdata/subdir", 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/testdata/file.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/testdata/subdir/file.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tests := []struct {
		pattern string
		want    string
	}{
		{tmpDir + "/testdata/*.txt", tmpDir + "/testdata"},
		{tmpDir + "/testdata/**/*.txt", tmpDir + "/testdata"},
		{tmpDir + "/testdata/file.txt", tmpDir + "/testdata"},
		{tmpDir + "/testdata/subdir/file.txt", tmpDir + "/testdata/subdir"},
		{tmpDir + "/testdata/subdir/*.txt", tmpDir + "/testdata/subdir"},
		{tmpDir + "/testdata/subdir/**/*.txt", tmpDir + "/testdata/subdir"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := getRootFromPattern(tt.pattern); got != tt.want {
				t.Errorf("getRootFromPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWatchPaths(t *testing.T) {
	TestDir := t.TempDir()
	if err := os.MkdirAll(TestDir+"/testdata/subdir", 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(TestDir+"/testdata/file.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(TestDir+"/testdata/subdir/file.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	tests := []struct {
		includePaths []string
		want         []string
	}{
		{[]string{TestDir + "/testdata/*.txt", TestDir + "/testdata/**/*.txt"}, []string{TestDir + "/testdata"}},
		{[]string{TestDir + "/testdata/file.txt", TestDir + "/testdata/subdir/file.txt"}, []string{TestDir + "/testdata", TestDir + "/testdata/subdir"}},
		{[]string{TestDir + "/testdata/subdir/*.txt", TestDir + "/testdata/subdir/**/*.txt"}, []string{TestDir + "/testdata/subdir"}},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			w := &Watcher{}
			t.Logf("includePaths: %v", tt.includePaths)
			got := w.getWatchPaths(tt.includePaths)
			if !equalStringSlices(got, tt.want) {
				t.Errorf("getWatchPaths() = %v, want %v", got, tt.want)
			}
			t.Logf("result: %v, want: %v", got, tt.want)
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int)
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		if seen[v] == 0 {
			return false
		}
		seen[v]--
	}
	return true
}

func TestIsStorgeFileIgnoresConfiguredStorageFiles(t *testing.T) {
	dataDir := t.TempDir()
	w := newTestWatcher(dataDir, nil, nil)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "系统文件", path: filepath.Join(dataDir, "filesystem.json"), want: true},
		{name: "进程快照", path: filepath.Join(dataDir, "process.json"), want: true},
		{name: "可疑文件列表", path: filepath.Join(dataDir, "dubious_files.json"), want: true},
		{name: "可疑进程列表", path: filepath.Join(dataDir, "dubious_processes.json"), want: true},
		{name: "未配置的存储文件", path: filepath.Join(dataDir, "other.json"), want: false},
		{name: "数据目录外的相同文件名", path: filepath.Join(t.TempDir(), "filesystem.json"), want: false},
		{name: "相同前缀在数据目录外", path: filepath.Join(dataDir+"-backup", "filesystem.json"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := w.isStorageFile(tt.path); got != tt.want {
				t.Fatalf("isStorageFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	dataDir := t.TempDir()
	includeRoot := t.TempDir()
	excludeRoot := filepath.Join(includeRoot, "cache")
	w := newTestWatcher(dataDir, []string{
		filepath.Join(includeRoot, "**"),
	}, []string{
		filepath.Join(excludeRoot, "**"),
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "配置的存储文件", path: filepath.Join(dataDir, "filesystem.json"), want: true},
		{name: "被排除的绝对通配符", path: filepath.Join(excludeRoot, "file.txt"), want: true},
		{name: "被包含的绝对通配符", path: filepath.Join(includeRoot, "app", "main.go"), want: false},
		{name: "未被包含", path: filepath.Join(t.TempDir(), "file.txt"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := w.shouldIgnore(tt.path); got != tt.want {
				t.Fatalf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestAddPathAddsOnlyIncludedDirectories(t *testing.T) {
	logger.Log = zap.NewNop()

	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	excludeDir := filepath.Join(root, "cache")
	allowedFile := filepath.Join(root, "app", "main.go")
	allowedDir := filepath.Dir(allowedFile)
	excludedFile := filepath.Join(excludeDir, "ignored.txt")
	storageFile := filepath.Join(dataDir, "filesystem.json")

	for _, dir := range []string{allowedDir, excludeDir, dataDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", dir, err)
		}
	}

	for _, path := range []string{allowedFile, excludedFile, storageFile} {
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}

	w, err := NewWatcher(&config.Config{
		Scanner: config.ScannerConfig{
			File: config.FileScannerConfig{
				IncludePaths: []string{filepath.Join(root, "**")},
				ExcludePaths: []string{filepath.Join(excludeDir, "**")},
			},
		},
		Storage: config.StorageConfig{
			DataDir:                dataDir,
			FileSystemFile:         "filesystem.json",
			ProcessSystemFile:      "process.json",
			DubiousFileListFile:    "dubious_files.json",
			DubiousProcessListFile: "dubious_processes.json",
		},
	})
	if err != nil {
		t.Fatalf("NewWatcher(): %v", err)
	}
	t.Cleanup(func() {
		_ = w.fsnWatcher.Close()
	})

	w.addPath(root)
	watched := w.fsnWatcher.WatchList()

	for _, path := range []string{root, allowedDir, dataDir} {
		if !containsString(watched, path) {
			t.Fatalf("WatchList() = %v, want included directory %q", watched, path)
		}
	}

	for _, path := range []string{allowedFile, excludeDir, excludedFile, storageFile} {
		t.Logf("Checking that %q is not in watch list", path)
		if containsString(watched, path) {
			t.Fatalf("WatchList() = %v, did not want %q", watched, path)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func newTestWatcher(dataDir string, includePaths, excludePaths []string) *Watcher {
	logger.Log = zap.NewNop()

	return &Watcher{
		cfg: &config.Config{
			Scanner: config.ScannerConfig{
				File: config.FileScannerConfig{
					IncludePaths: includePaths,
					ExcludePaths: excludePaths,
				},
			},
			Storage: config.StorageConfig{
				DataDir:                dataDir,
				FileSystemFile:         "filesystem.json",
				ProcessSystemFile:      "process.json",
				DubiousFileListFile:    "dubious_files.json",
				DubiousProcessListFile: "dubious_processes.json",
			},
		},
	}
}
