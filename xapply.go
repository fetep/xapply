package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/fetep/xapply/dicer"
	flag "github.com/spf13/pflag"
)

var (
	flgShell    = flag.StringP("shell", "S", os.Getenv("SHELL"), "shell to run commands with")
	flgNoop     = flag.BoolP("noop", "n", false, "print what would be run to stdout instead of running it")
	flgVerbose  = flag.BoolP("verbose", "v", false, "print command to stdout as it is running")
	flgVerboseX = flag.BoolP("verbosex", "x", false, "print command to stderr as it is running")
)

func apply(tmpl string, input string) error {
	cmdLine, err := dicer.Expand(tmpl, []string{input})
	if err != nil {
		panic(err)
	}

	if *flgNoop {
		fmt.Printf("%s\n", cmdLine)
		return nil // nothing further needed
	}

	if *flgVerbose {
		fmt.Printf("%s\n", cmdLine)
	}

	if *flgVerboseX {
		fmt.Fprintf(os.Stderr, "%s\n", cmdLine)
	}

	cmd := exec.Command(*flgShell, "-c", cmdLine)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stdoutReader := bufio.NewReader(stdout)
	cmd.Start()

	for {
		line, err := stdoutReader.ReadBytes('\n')
		if err != nil {
			break
		}

		fmt.Printf(string(line))
	}

	cmd.Wait()

	return nil
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatalf("missing command")
	}

	cmdTmpl := flag.Args()[0]
	args := flag.Args()[1:]

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
