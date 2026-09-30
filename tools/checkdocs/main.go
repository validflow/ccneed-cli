package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// main 执行仓库注释门禁，失败时返回非零退出码。
func main() {
	violations, err := scan(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, violation := range violations {
		fmt.Fprintln(os.Stderr, violation)
	}
	if len(violations) > 0 {
		os.Exit(1)
	}
	fmt.Println("Go documentation contract passed")
}

// scan 检查目录内所有非测试 Go 文件，不依赖 Git 的暂存状态。
//
// Args:
//
//	root: 仓库根目录。
//
// Returns:
//
//	注释违规说明及遍历或解析错误。
//
// Errors:
//
//	文件不可读、目录不可遍历或 Go 语法错误时返回错误。
func scan(root string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && skipDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		positions := token.NewFileSet()
		file, parseErr := parser.ParseFile(positions, path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}
		for _, violation := range checkFile(positions, file) {
			violations = append(violations, path+": "+violation)
		}
		return nil
	})
	return violations, err
}

// skipDirectory 排除版本管理、依赖及构建产物目录。
//
// Args:
//
//	name: 目录名，不含完整路径。
//
// Returns:
//
//	不应扫描该目录时为 true。
func skipDirectory(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "bin" || name == "dist"
}
