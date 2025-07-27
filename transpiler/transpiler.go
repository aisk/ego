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
	if results == nil || len(results.List) == 0 {
		return nil, fmt.Errorf("try expression used in function that does not return an error")
	}

	fields := results.List
	var resultsExpr []ast.Expr

	// Check if the last parameter is an error type
	lastField := fields[len(fields)-1]
	if ident, ok := lastField.Type.(*ast.Ident); !ok || ident.Name != "error" {
		return nil, fmt.Errorf("try expression used in function that does not return an error")
	}

	// Generate empty values for all parameters
	for _, field := range fields {
		expr, err := genEmptyValueExpr(field)
		if err != nil {
			return nil, err
		}
		resultsExpr = append(resultsExpr, expr)
	}

	// Replace the last parameter (which is error) with "err"
	resultsExpr[len(resultsExpr)-1] = &ast.Ident{Name: "err"}

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
		case *ast.IfStmt:
			// Handle if statements with TryExpr in condition
			// Specifically handle the pattern: if f()? > 0 { ... }
			if tryExpr := findTopLevelTryExpr(x.Cond); tryExpr != nil {
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

				// Replace the TryExpr with a variable
				newCond := replaceTryExpr(x.Cond, tryExpr, "result")

				// Create the new if statement with init
				newIf := &ast.IfStmt{
					Init: &ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.Ident{Name: "result"},
							&ast.Ident{Name: "err"},
						},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{tryExpr.X},
					},
					Cond: &ast.BinaryExpr{
						X:  &ast.Ident{Name: "err"},
						Op: token.NEQ,
						Y:  &ast.Ident{Name: "nil"},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{Results: results},
						},
					},
					Else: &ast.IfStmt{
						Cond: newCond,
						Body: x.Body,
						Else: x.Else,
					},
				}

				// Handle init statement if it exists
				if x.Init != nil {
					// If there's an init statement, we need to create a new if statement
					// that incorporates both the init and the error handling
					newIf.Init = x.Init
				}

				c.Replace(newIf)
			}
		}

		return true
	})

	if transpileError != nil {
		return transpileError
	}

	return format.Node(output, fset, file)
}

// containsTryExpr checks if an expression contains any TryExpr nodes
func containsTryExpr(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if _, ok := n.(*ast.TryExpr); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

// transformTryExpr transforms an expression containing TryExpr into a new expression
// and returns the assignments needed to handle error checking
func transformTryExpr(expr ast.Expr, prefix string, results []ast.Expr) (ast.Expr, []ast.Stmt) {
	var assignments []ast.Stmt

	// Create a map to track TryExpr replacements
	tryReplacements := make(map[*ast.TryExpr]string)

	// First pass: collect all TryExpr nodes and create replacement variables
	ast.Inspect(expr, func(n ast.Node) bool {
		if tryExpr, ok := n.(*ast.TryExpr); ok {
			if _, exists := tryReplacements[tryExpr]; !exists {
				varName := fmt.Sprintf("%s%d", prefix, len(tryReplacements))
				tryReplacements[tryExpr] = varName

				// Create assignment statement
				assign := &ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.Ident{Name: varName},
						&ast.Ident{Name: "err"},
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{tryExpr.X},
				}

				// Create error check
				errCheck := &ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  &ast.Ident{Name: "err"},
						Op: token.NEQ,
						Y:  &ast.Ident{Name: "nil"},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{Results: results},
						},
					},
				}

				assignments = append(assignments, assign, errCheck)
			}
		}
		return true
	})

	// Second pass: create a new expression with TryExpr replaced
	// We need to create a deep copy of the expression
	// For now, we'll use a simple approach by walking and replacing
	var transformExpr func(ast.Expr) ast.Expr
	transformExpr = func(e ast.Expr) ast.Expr {
		switch v := e.(type) {
		case *ast.TryExpr:
			if varName, exists := tryReplacements[v]; exists {
				return &ast.Ident{Name: varName}
			}
			return v
		case *ast.BinaryExpr:
			return &ast.BinaryExpr{
				X:  transformExpr(v.X),
				Op: v.Op,
				Y:  transformExpr(v.Y),
			}
		case *ast.UnaryExpr:
			return &ast.UnaryExpr{
				Op: v.Op,
				X:  transformExpr(v.X),
			}
		case *ast.ParenExpr:
			return &ast.ParenExpr{X: transformExpr(v.X)}
		case *ast.CallExpr:
			fun := transformExpr(v.Fun)
			args := make([]ast.Expr, len(v.Args))
			for i, arg := range v.Args {
				args[i] = transformExpr(arg)
			}
			return &ast.CallExpr{Fun: fun, Args: args}
		default:
			return v
		}
	}

	newExpr := transformExpr(expr)
	return newExpr, assignments
}

// findTopLevelTryExpr finds the top-level TryExpr in an expression
func findTopLevelTryExpr(expr ast.Expr) *ast.TryExpr {
	switch v := expr.(type) {
	case *ast.TryExpr:
		return v
	case *ast.BinaryExpr:
		// Check both sides of binary expression
		if left := findTopLevelTryExpr(v.X); left != nil {
			return left
		}
		return findTopLevelTryExpr(v.Y)
	case *ast.ParenExpr:
		return findTopLevelTryExpr(v.X)
	case *ast.UnaryExpr:
		return findTopLevelTryExpr(v.X)
	case *ast.CallExpr:
		// Check function and arguments
		if fun := findTopLevelTryExpr(v.Fun); fun != nil {
			return fun
		}
		for _, arg := range v.Args {
			if try := findTopLevelTryExpr(arg); try != nil {
				return try
			}
		}
	}
	return nil
}

// replaceTryExpr replaces a specific TryExpr with a variable name
func replaceTryExpr(expr ast.Expr, oldExpr *ast.TryExpr, varName string) ast.Expr {
	if expr == oldExpr {
		return &ast.Ident{Name: varName}
	}

	switch v := expr.(type) {
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			X:  replaceTryExpr(v.X, oldExpr, varName),
			Op: v.Op,
			Y:  replaceTryExpr(v.Y, oldExpr, varName),
		}
	case *ast.ParenExpr:
		return &ast.ParenExpr{X: replaceTryExpr(v.X, oldExpr, varName)}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{Op: v.Op, X: replaceTryExpr(v.X, oldExpr, varName)}
	case *ast.CallExpr:
		fun := replaceTryExpr(v.Fun, oldExpr, varName)
		args := make([]ast.Expr, len(v.Args))
		for i, arg := range v.Args {
			args[i] = replaceTryExpr(arg, oldExpr, varName)
		}
		return &ast.CallExpr{Fun: fun, Args: args}
	}
	return expr
}
