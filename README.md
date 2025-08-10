# ego 

![logo](https://dev-to-uploads.s3.amazonaws.com/uploads/articles/tszc13irysyrnvg34lzp.png)

`ego` means **E**rror can be handled implicitly in **GO**.

It's an experimental *toy* Go preprocessor/transpiler that introduces the `?` operator for more concise error handling. It aims to reduce boilerplate code by replacing verbose error checks with a single character.

For example, `ego` transforms this:

```go
s := hello()?
```

Into this:

```go
s, err := hello()
if err != nil {
    return err
}
```

## Installation

```sh
$ go install github.com/aisk/ego@latest
```

## Usage

Create a file named `hello.ego` with the following content:

```go
package main

import (
	"io"
	"os"
)

func hello() error {
	f := os.Open("hello.ego")?
	defer f.Close()
	s := io.ReadAll(f)?
	println(string(s))
	return nil
}

func main() {
	hello()
}
```

`ego` supports multiple ways to specify transpile targets:

```sh
# Transpile from stdin
$ cat hello.ego | ego > hello.go

# Transpile specific files
$ ego hello.ego

# Transpile all .ego files in a directory (non-recursive)
$ ego .

# Transpile all .ego files recursively
$ ego ./...

# Transpile all .ego files in current directory recursively
$ ego ...
```

The transpiled `hello.go` will contain:

```go
package main

import (
	"io"
	"os"
)

func hello() error {
	f, err := os.Open("hello.ego")
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	println(string(s))
	return nil
}

func main() {
	hello()
}

```

## Design Principles

The goal of this project is to design a Go language extension with more syntactic sugar, implemented as a preprocessor. The precompiled result is **completely standard Go code**, indistinguishable from hand-written Go code.

**Core Design Philosophy: Zero Lock-in**

- If you decide this project isn't suitable, you can simply delete all `.ego` files and continue development with the original Go code
- If your team doesn't want to introduce ego, you can edit `.ego` files locally and commit the generated standard Go code to your version control server
- The precompiled Go code has no runtime dependencies or special libraries

**Design Constraints**

To achieve zero lock-in, there are some intentional limitations:

1. **The ? operator does not support method chaining** - For example, `a()?.b()?` is not supported because it would require introducing intermediate variables, and it's difficult to provide reasonable names for these variables
2. **Currently does not support the ? operator in for loop initialization statements** - For example, `for item := range getItems()?` is not yet supported. This limitation may be removed in the future
3. **When discarding return values, functions must have only an error return** - When you don't accept any return values from a function (i.e., when using `f()?`), the function must have exactly one return value of type `error`. If a function returns multiple values (e.g., `func f() (int, error)`), you need to use `_` to discard the non-error return values: `_ = f()?`

These constraints ensure the generated Go code remains clean, readable, and identical to hand-written code.

## TODO

- [ ] Implement a Go-compatible command-line tool that supports all Go flags and commands, with the only difference being that it preprocesses all `.ego` files to `.go` files before running compile or test operations
