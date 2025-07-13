# ego 

![logo](https://dev-to-uploads.s3.amazonaws.com/uploads/articles/tszc13irysyrnvg34lzp.png)

`ego` means **E**rror can be handled implicitly in **GO**.

It's an experimental *toy* Go transpiler that introduces the `?` operator for more concise error handling. It aims to reduce boilerplate code by replacing verbose error checks with a single character.

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

Run the `ego` transpiler and save the output to a `.go` file:

```sh
$ cat hello.ego | ego > hello.go
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