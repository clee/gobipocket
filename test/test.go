package main

import (
	"fmt"
	"flag"
	"github.com/clee/gobipocket"
)


func main() {
	flag.Parse()
	for _, path := range flag.Args() {
		m, err := mobipocket.Open(path)
		if err != nil {
			panic(err)
		}

		for k, v := range m.Metadata {
			fmt.Printf("%s: %s\n", k, v)
		}
	}
}
