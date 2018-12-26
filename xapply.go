package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/fetep/xapply/dicer"
)

func apply(tmpl string, input string) {
	output, err := dicer.Expand(tmpl, []string{input})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", output)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("need a command")
	}

	cmdTmpl, args := os.Args[1], os.Args[2:]
	for _, input := range args {
		if input == "-" {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				apply(cmdTmpl, scanner.Text())
			}
		} else {
			apply(cmdTmpl, input)
		}
	}
}
