package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/fetep/xapply/dicer"
)

func apply(dicerTmpl dicer.Template, input string) {
	output, err := dicerTmpl.Expand([]string{input})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", output)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("need a command")
	}

	cmd, args := os.Args[1], os.Args[2:]
	dicerTmpl := dicer.NewTemplate(cmd)
	for _, input := range args {
		if input == "-" {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				apply(dicerTmpl, scanner.Text())
			}
		} else {
			apply(dicerTmpl, input)
		}
	}
}
