package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var exitizer = &analysis.Analyzer{
	Name: "exitizer",
	Doc:  "check for os.Exit calls in main function",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	checkForExitCalls := func(node ast.Node) {
		ast.Inspect(node, func(node ast.Node) bool {
			callExpr, isCallExpr := node.(*ast.CallExpr)
			if isCallExpr {
				selectorExpr, isSelectorExpr := callExpr.Fun.(*ast.SelectorExpr)
				if !isSelectorExpr {
					return false
				}

				x, isX := selectorExpr.X.(*ast.Ident)
				if !isX {
					return false
				}

				if x.Name == "os" && selectorExpr.Sel.Name == "Exit" {
					pass.Reportf(callExpr.Pos(), "os.Exit call")
					return false
				}
			}

			return true
		})
	}
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if !strings.HasSuffix(filename, ".go") {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			funcdecl, ok := node.(*ast.FuncDecl)
			if ok && funcdecl.Name.Name == "main" {
				checkForExitCalls(funcdecl)
				return false
			}

			return true
		})
	}

	return nil, nil
}
