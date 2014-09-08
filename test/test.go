package main

import (
	"fmt"
	"flag"
	"os"
	"runtime/pprof"
	"github.com/clee/gobipocket"
)


func main() {
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to file")
	flag.Parse()
	path := flag.Arg(0)
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	m, err := mobipocket.Open(path)
	if err != nil {
		panic(err)
	}

	for k, v := range m.Metadata {
		fmt.Printf("%s: %s\n", k, v)
	}
}
