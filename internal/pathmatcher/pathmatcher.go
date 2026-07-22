package pathmatcher

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// 判断路径是否包含通配符
func HasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[{")
}

// 展开为实际路径列表
func ExpandPaths(patterns []string) ([]string, error) {
	result := make([]string, 0)

	for _, pattern := range patterns {
		pattern = filepath.Clean(pattern)
		if !HasGlobMeta(pattern) {
			result = append(result, pattern)
			continue
		}

		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return nil, err
		}

		result = append(result, matches...)
	}

	return uniqueStrings(result), nil
}

func uniqueStrings(input []string) []string {
	result := make([]string, 0, len(input))
	uniqueMap := make(map[string]struct{})
	for _, str := range input {
		if str == "" {
			continue
		}

		if _, exists := uniqueMap[str]; exists {
			continue
		}

		result = append(result, str)
		uniqueMap[str] = struct{}{}
	}

	return result
}

// 判断路径是否匹配给定的模式列表， Glob/前缀匹配
func MatchPath(pattern string, path string) bool {
	pattern = filepath.Clean(pattern)
	path = filepath.Clean(path)

	if HasGlobMeta(pattern) {
		matched, err := doublestar.PathMatch(pattern, path)
		if err != nil {
			return false
		}
		return matched
	}

	return pattern == path || strings.HasPrefix(path, pattern+string(filepath.Separator))
}

func IsMatchAnyPath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if MatchPath(pattern, path) {
			return true
		}
	}
	return false
}

// glob提取根目录
func ExtractRootFromGlob(pattern string) string {
	pattern = filepath.Clean(pattern)

	if !HasGlobMeta(pattern) {
		return pattern
	}

	parts := strings.Split(pattern, string(filepath.Separator))

	for i, part := range parts {
		if strings.ContainsAny(part, "*?[{") {
			if i == 0 {
				return "."
			}

			return filepath.Join(parts[:i]...)
		}
	}
	return pattern
}

func ExtractWalkRoots(patterns []string) []string {
	roots := make([]string, 0)
	seen := make(map[string]struct{})

	for _, pattern := range patterns {
		root := ExtractRootFromGlob(pattern)
		if _, exists := seen[root]; !exists {
			seen[root] = struct{}{}
			roots = append(roots, root)
		}
	}

	return roots
}
