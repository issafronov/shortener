// Package noosexit запрещает использование os.Exit в функции main пакета main.
package noosexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "запрещает прямое использование os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Recv != nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				obj := pass.TypesInfo.Uses[pkg]
				if obj == nil {
					return true
				}

				if obj.Pkg() != nil && obj.Pkg().Path() == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "использование os.Exit в main запрещено, используйте return из main")
				}

				return true
			})
		}
	}
	return nil, nil
}
