package main

import (
	"os"

	"github.com/aisk/ego/ast"
	"github.com/aisk/ego/astutil"
	"github.com/aisk/ego/format"
	"github.com/aisk/ego/parser"
	"github.com/aisk/ego/token"
)

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", os.Stdin, 0)
	if err != nil {
		panic(err)
	}

	astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		n := c.Node()
		switch x := n.(type) {
		case *ast.AssignStmt:
			// Handle err := f()?
			rhs, ok := x.Rhs[0].(*ast.TryExpr)
			if !ok {
				break
			}

			x.Rhs[0] = rhs.X
			x.Lhs = append(x.Lhs, &ast.Ident{Name: "err"})

			c.InsertAfter(&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  &ast.Ident{Name: "err"},
					Op: token.NEQ,
					Y:  &ast.Ident{Name: "nil"},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{
							Results: []ast.Expr{
								&ast.Ident{Name: "err"},
							},
						},
					},
				}})
		}

		return true
	})

	format.Node(os.Stdout, fset, file)
}
