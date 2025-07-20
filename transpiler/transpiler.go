package transpiler

import (
	"errors"
	"fmt"
	"io"

	"github.com/aisk/ego/ast"
	"github.com/aisk/ego/astutil"
	"github.com/aisk/ego/containers"
	"github.com/aisk/ego/format"
	"github.com/aisk/ego/parser"
	"github.com/aisk/ego/token"
)

var fstack containers.Stack[*ast.FuncType]

func preVisit(c *astutil.Cursor) bool {
	// Push FuncType to a stack for find the enclosing one.
	n := c.Node()
	switch x := n.(type) {
	case *ast.FuncDecl:
		fstack.Push(x.Type)
	case *ast.FuncLit:
		fstack.Push(x.Type)
	}
	return true
}

func getEnclosingFuncType() (*ast.FuncType, error) {
	ftype, exist := fstack.Peek()
	if !exist {
		return nil, errors.New("no enclosing function")
	}
	return ftype, nil
}

func genEmptyValueExpr(field *ast.Field) (ast.Expr, error) {
	if ident, ok := field.Type.(*ast.Ident); ok {
		switch field.Type.(*ast.Ident).Name {
		case "error":
			return &ast.Ident{Name: "nil"}, nil
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "uintptr", "rune", "byte":
			return &ast.BasicLit{Kind: token.INT, Value: "0"}, nil
		case "bool":
			return &ast.Ident{Name: "false"}, nil
		case "string":
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}, nil
		default:
			return &ast.StarExpr{
				X: &ast.CallExpr{
					Fun: &ast.Ident{Name: "new"},
					Args: []ast.Expr{
						&ast.Ident{Name: ident.Name},
					},
				},
			}, nil
		}
	} else if selector, ok := field.Type.(*ast.SelectorExpr); ok {
		return &ast.StarExpr{
			X: &ast.CallExpr{
				Fun: &ast.Ident{Name: "new"},
				Args: []ast.Expr{
					&ast.SelectorExpr{
						X:   &ast.Ident{Name: selector.X.(*ast.Ident).Name},
						Sel: &ast.Ident{Name: selector.Sel.Name},
					},
				},
			},
		}, nil
	} else if _, ok := field.Type.(*ast.StarExpr); ok {
		return &ast.Ident{Name: "nil"}, nil
	} else {
		return nil, fmt.Errorf("unhandled result type: %T", field.Type)
	}
}

func genResults(results *ast.FieldList) ([]ast.Expr, error) {
	if results == nil {
		return nil, fmt.Errorf("try expression used in function that does not return an error")
	}
	fields := results.List
	var resultsExpr []ast.Expr
	hasError := false
	for _, field := range fields {
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "error" {
			hasError = true
		}
		expr, err := genEmptyValueExpr(field)
		if err != nil {
			return nil, err
		}
		resultsExpr = append(resultsExpr, expr)
	}

	if !hasError {
		return nil, fmt.Errorf("try expression used in function that does not return an error")
	}

	for i := len(fields) - 1; i >= 0; i-- {
		field := fields[i]
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "error" {
			resultsExpr[i] = &ast.Ident{Name: "err"}
			break
		}
	}

	return resultsExpr, nil
}

func getReaderFileName(reader io.Reader) string {
	filename := "*unknown*"
	if f, ok := reader.(interface{ Name() string }); ok {
		filename = f.Name()
	}
	return filename
}

func Transpile(input io.Reader, output io.Writer) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, getReaderFileName(input), input, 0)
	if err != nil {
		return err
	}

	// ast.Print(fset, file)

	var transpileError error

	astutil.Apply(file, preVisit, func(c *astutil.Cursor) bool {
		n := c.Node()
		switch x := n.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			// Pop the FuncType stack.
			fstack.Pop()
		case *ast.AssignStmt:
			// Handle err := f()?
			rhs, ok := x.Rhs[0].(*ast.TryExpr)
			if !ok {
				break
			}
			enclosingFunc, err := getEnclosingFuncType()
			if err != nil {
				transpileError = fmt.Errorf("%s: %v", fset.Position(x.Pos()), err)
				return false
			}

			results, err := genResults(enclosingFunc.Results)
			if err != nil {
				transpileError = fmt.Errorf("%s: %v", fset.Position(x.Pos()), err)
				return false
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
							Results: results,
						},
					},
				}})
		case *ast.ExprStmt:
			// Handle f()?
			tryX, ok := x.X.(*ast.TryExpr)
			if !ok {
				break
			}

			enclosingFunc, err := getEnclosingFuncType()
			if err != nil {
				transpileError = fmt.Errorf("%s: %v", fset.Position(x.Pos()), err)
				return false
			}

			results, err := genResults(enclosingFunc.Results)
			if err != nil {
				transpileError = fmt.Errorf("%s: %v", fset.Position(x.Pos()), err)
				return false
			}

			c.Replace(&ast.IfStmt{
				Init: &ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.Ident{Name: "err"},
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						tryX.X,
					},
				},
				Cond: &ast.BinaryExpr{
					X:  &ast.Ident{Name: "err"},
					Op: token.NEQ,
					Y:  &ast.Ident{Name: "nil"},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{
							Results: results,
						},
					},
				},
			})
		}

		return true
	})

	if transpileError != nil {
		return transpileError
	}

	return format.Node(output, fset, file)
}
