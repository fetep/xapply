package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/fetep/xapply/dicer"
	flag "github.com/spf13/pflag"
)

var (
	flgFile     = flag.BoolP("file", "f", false, "treat arguments as filenames to read inputs from")
	flgShell    = flag.StringP("shell", "S", os.Getenv("SHELL"), "shell to run commands with")
	flgNoop     = flag.BoolP("noop", "n", false, "print what would be run to stdout instead of running it")
	flgParallel = flag.IntP("parallel", "P", 1, "the number of processes to run in parallel")
	flgVerbose  = flag.BoolP("verbose", "v", false, "print command to stdout as it is running")
	flgVerboseX = flag.BoolP("verbosex", "x", false, "print command to stderr as it is running")
)

var workers sync.WaitGroup

// apply runs dicer on tmpl with the provided input and runs the command.
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

type jobData struct {
	cmdTmpl  string
	input    string
	finished bool // set to true when the worker should finish.
}

// worker pulls inputs from the jobs queue and calls apply.
// if the job has finished set, there is no more input and we are done.
func worker(jobs <-chan jobData) {
	defer workers.Done()

	for job := range jobs {
		if job.finished {
			break
		}

		apply(job.cmdTmpl, job.input)
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatalf("missing command")
	}

	if *flgParallel < 1 {
		log.Fatalf("-P must be at least 1")
	}

	cmdTmpl := flag.Args()[0]
	args := flag.Args()[1:]
	jobs := make(chan jobData)

	// Fire up workers
	workers.Add(*flgParallel)
	for i := 0; i < *flgParallel; i++ {
		go worker(jobs)
	}

	// Generate jobs
	for _, input := range args {

		if *flgFile {
			var file *os.File

			if input == "-" {
				file = os.Stdin
			} else {
				var err error
				file, err = os.Open(input)
				if err != nil {
					log.Printf("%s: error opening: %s", input, err)
					continue
				}
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				jobs <- jobData{cmdTmpl, scanner.Text(), false}
			}
		} else {
			jobs <- jobData{cmdTmpl, input, false}
		}
	}

	// When workers all finish their current job, they'll get the finish message
	for i := 0; i < *flgParallel; i++ {
		jobs <- jobData{"", "", true}
	}

	workers.Wait()
}
