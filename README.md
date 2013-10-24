# mobipocket (for golang)

This is a library designed to make it possible to read some of
the metadata from a Mobipocket-formatted file (or Kindle ebook)
using the Go programming language. The current API is limited
to just the Open and Metadata calls; you can use them like this:

```go
package main

import (
	"fmt"
	"flag"
	"github.com/clee/gobipocket"
)

func main() {
	flag.Parse()
	path := flag.Arg(0)
	m, err := mobipocket.Open(path)
	if err != nil {
		panic(err)
	}

	for k, v := range m.Metadata {
		fmt.Printf("%s: %s\n", k, v)
	}
}
```
