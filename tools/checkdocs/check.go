// Package main 检查仓内 Go 源码的注释契约。
//
// 用法：在仓库根目录执行 go run ./tools/checkdocs。
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

var issuePattern = regexp.MustCompile(`\bCCN-\d+\b`)

// checkFile 校验一个已解析文件中的声明注释。
//
// Args:
//
//	positions: 源码位置表。
//	file: 保留注释的语法树。
//
// Returns:
//
//	所有违规说明；空切片表示通过。
func checkFile(positions *token.FileSet, file *ast.File) []string {
	var violations []string
	for _, group := range file.Comments {
		if issuePattern.MatchString(group.Text()) {
			violations = append(violations, fmt.Sprintf("%s: issue identifier in comment", positions.Position(group.Pos())))
		}
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			violations = append(violations, checkFunction(declaration)...)
		case *ast.GenDecl:
			violations = append(violations, checkTypes(declaration)...)
		}
	}
	return violations
}

// checkFunction 根据函数签名校验注释所需段落。
//
// Args:
//
//	function: 具名函数或方法声明。
//
// Returns:
//
//	该函数的注释违规说明。
func checkFunction(function *ast.FuncDecl) []string {
	name := function.Name.Name
	violations := checkSummary(name, function.Doc)
	if function.Doc == nil {
		return violations
	}
	documentation := function.Doc.Text()
	if function.Type.Params.NumFields() > 0 && !hasSection(documentation, "Args:") {
		violations = append(violations, name+": missing Args:")
	}
	if function.Type.Results.NumFields() > 0 && !hasSection(documentation, "Returns:") {
		violations = append(violations, name+": missing Returns:")
	}
	if returnsError(function.Type) && !hasSection(documentation, "Errors:") {
		violations = append(violations, name+": missing Errors:")
	}
	return violations
}

// checkTypes 检查独立或分组类型声明的注释。
//
// Args:
//
//	declaration: 可能包含类型声明的通用声明。
//
// Returns:
//
//	类型注释违规说明；非类型声明返回空切片。
func checkTypes(declaration *ast.GenDecl) []string {
	var violations []string
	for _, specification := range declaration.Specs {
		typeSpec, ok := specification.(*ast.TypeSpec)
		if !ok {
			continue
		}
		documentation := typeSpec.Doc
		if documentation == nil && len(declaration.Specs) == 1 {
			documentation = declaration.Doc
		}
		violations = append(violations, checkSummary(typeSpec.Name.Name, documentation)...)
	}
	return violations
}

// checkSummary 检查注释是否存在并以完整符号名开头。
//
// Args:
//
//	name: 声明的符号名。
//	documentation: 紧邻声明的注释，可能为空。
//
// Returns:
//
//	首句违规说明。
func checkSummary(name string, documentation *ast.CommentGroup) []string {
	if documentation == nil {
		return []string{name + ": missing doc comment"}
	}
	words := strings.Fields(documentation.Text())
	if len(words) == 0 || words[0] != name {
		return []string{name + ": comment must start with symbol name"}
	}
	return nil
}

// hasSection 查找独占一行的段落标题。
//
// Args:
//
//	documentation: 完整注释文本。
//	section: 所需段落标题。
//
// Returns:
//
//	标题存在时为 true。
func hasSection(documentation, section string) bool {
	for line := range strings.SplitSeq(documentation, "\n") {
		if strings.TrimSpace(line) == section {
			return true
		}
	}
	return false
}

// returnsError 检查返回值是否包含内建 error。
//
// Args:
//
//	signature: 函数签名。
//
// Returns:
//
//	直接返回内建 error 时为 true。
func returnsError(signature *ast.FuncType) bool {
	if signature.Results == nil {
		return false
	}
	for _, field := range signature.Results.List {
		identifier, ok := field.Type.(*ast.Ident)
		if ok && identifier.Name == "error" {
			return true
		}
	}
	return false
}
