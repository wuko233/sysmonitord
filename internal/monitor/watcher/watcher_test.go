package watcher

import (
	"os"
	"testing"
)

func TestGetRootFromPattern(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(tmpDir+"/testdata/subdir", 0755)
	os.WriteFile(tmpDir+"/testdata/file.txt", []byte("test"), 0644)
	os.WriteFile(tmpDir+"/testdata/subdir/file.txt", []byte("test"), 0644)

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
	os.MkdirAll(TestDir+"/testdata/subdir", 0755)
	os.WriteFile(TestDir+"/testdata/file.txt", []byte("test"), 0644)
	os.WriteFile(TestDir+"/testdata/subdir/file.txt", []byte("test"), 0644)

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
