package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// exitCheckAnalyzer reports direct calls to os.Exit inside main.main.
var exitCheckAnalyzer = &analysis.Analyzer{
	Name: "noexitmain",
	Doc:  "report direct calls to os.Exit in the main function of package main",
	Run: func(pass *analysis.Pass) (any, error) {
		if pass.Pkg.Name() != "main" || strings.HasSuffix(pass.Pkg.Path(), ".test") {
			return nil, nil
		}
		for _, file := range pass.Files {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Name.Name != "main" || function.Body == nil {
					continue
				}
				ast.Inspect(function.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					callee := call.Fun
					for {
						parenthesized, ok := callee.(*ast.ParenExpr)
						if !ok {
							break
						}
						callee = parenthesized.X
					}
					var name *ast.Ident
					switch expression := callee.(type) {
					case *ast.SelectorExpr:
						name = expression.Sel
					case *ast.Ident:
						name = expression
					}
					if name == nil {
						return true
					}
					function, ok := pass.TypesInfo.Uses[name].(*types.Func)
					if ok && function.Pkg() != nil && function.Pkg().Path() == "os" && function.Name() == "Exit" {
						pass.Reportf(call.Pos(), "direct call to os.Exit from main")
					}
					return true
				})
			}
		}
		return nil, nil
	},
}
