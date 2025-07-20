# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ego** is an experimental Go preprocessor/transpiler that introduces the `?` operator for more concise error handling. It transforms Go source code with the `?` operator into standard Go code with proper error handling.

## Architecture

The project is structured into several key packages that work together to parse, transform, and format Go code:

### Core Packages

- **parser/**: Extends the standard Go parser to support the `?` operator (TryExpr)
- **ast/**: Extends the standard Go AST to include TryExpr for the `?` operator
- **transpiler/**: Main transformation logic that converts `?` operators into standard Go error handling
- **printer/**: Code formatting and output generation
- **format/**: Source code formatting utilities
- **token/**: Token definitions and position tracking
- **scanner/**: Lexical analysis for Go source
- **astutil/**: AST manipulation utilities

### Key Components

1. **TryExpr**: New AST node type in `ast/ast.go:400-403` representing `expr?` syntax
2. **Transpiler**: The main transformation engine in `transpiler/transpiler.go` that:
   - Uses AST traversal to find TryExpr nodes
   - Generates appropriate error handling boilerplate
   - Maintains function context for return types
3. **Parser Extension**: Modified Go parser in `parser/parser.go` that recognizes `?` as TryExpr

## Development Commands

### Build
```bash
go build -o ego
```

### Test
```bash
# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./parser
```

### Usage
```bash
# Process .ego file to .go file
cat input.ego | ./ego > output.go

# Install globally
go install github.com/aisk/ego@latest
```

### Development Workflow

1. **Code Structure**: The project extends Go's standard library packages (go/ast, go/parser, etc.) with ego-specific functionality
2. **Testing**: Uses standard Go testing with testdata directories containing source files
3. **Error Handling**: The transpiler validates that `?` is only used in functions that return error types

## Key Files

- `main.go`: Entry point - reads from stdin, writes transpiled Go to stdout
- `transpiler/transpiler.go`: Core transformation logic with AST manipulation
- `ast/ast.go`: Extended AST node definitions including TryExpr
- `parser/parser.go`: Extended parser supporting the `?` operator
- `printer/printer.go`: Go code formatting and output

## Code Patterns

- **AST Manipulation**: Uses `astutil.Apply` for AST traversal and modification
- **Function Context**: Uses a stack (`containers.Stack`) to track enclosing function types for proper error return handling
- **Source Position**: Maintains accurate source positions for error reporting
- **Boilerplate Generation**: Automatically generates `if err != nil { return ... }` patterns
