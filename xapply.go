package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fetep/xapply/dicer"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("need a command")
	}

	cmd, args := os.Args[1], os.Args[2:]
	dicerTmpl := dicer.NewTemplate(cmd)
	for _, input := range args {
		diced, err := dicerTmpl.Expand([]string{input})
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", diced)
	}
}
