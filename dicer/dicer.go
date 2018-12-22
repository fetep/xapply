package dicer

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

// Template represents a dicer template.
type Template struct {
	runes []rune
}

// NewTemplate creates a new dicer template from a string.
func NewTemplate(tmpl string) Template {
	return Template{
		runes: []rune(tmpl),
	}
}

// Expand the dicer template given a set of strings mapping to %1, %2, etc.
func (t *Template) Expand(inputs []string) (string, error) {
	var out string

	for pos := 0; pos < len(t.runes); pos++ {
		char := t.runes[pos]
		//log.Printf("pos=%v char=%v", pos, char)

		// A '%' dicer escape followed by at least one character means we need to expand it.
		if char == '%' && pos < len(t.runes)-1 {
			peek := t.runes[pos+1]
			switch {
			case peek == '%':
				pos++
				out += "%"
				continue
			case peek == '[':
				dicerExpr, err := t.readUntil(']', pos+2)
				if err != nil {
					return "", err
				}

				pos += len(dicerExpr) + 2
				// TODO for now, we just allow dicerExpr to be the actual index number with no dicing.
				index, _ := strconv.ParseInt(dicerExpr, 10, 64)
				out += inputs[index-1]
				continue
			case t.isNumber(pos + 1):
				inputIndex, inputLength, err := t.readNumber(pos + 1)
				if err != nil {
					return "", err
				}
				if inputIndex > len(inputs) {
					return "", fmt.Errorf("template references out of bounds input index %d", inputIndex)
				}

				pos += inputLength
				out += inputs[inputIndex-1]
				continue
			}
		}

		out += string(char)
	}

	return out, nil
}

func (t *Template) String() string {
	return string(t.runes)
}

// Given a slice of runes with the first element is a [0-9], read runes as long as we get [0-9]s
// and return an int representation of these numbers along with the number of runes we consumed.
func (t *Template) readNumber(start int) (int, int, error) {
	var number string

	if start > len(t.runes)-1 {
		return 0, 0, fmt.Errorf("readNumber called with start=%d which is longer than runes (%d)", start, len(t.runes)-1)
	}

	for i := start; i < len(t.runes) && t.isNumber(i); i++ {
		number += string(t.runes[i])
	}

	if len(number) == 0 { // the rune at start isn't a number!
		return 0, 0, fmt.Errorf("readNumber called with start=%d, which is not an integer: %v", start, string(t.runes[start]))
	}

	i, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return int(i), len(number), nil
}

// Given a slice of runes, read runes until one matches end and return a string up to but not including end
func (t *Template) readUntil(end rune, start int) (string, error) {
	for i := start; i < len(t.runes); i++ {
		if t.runes[i] == end {
			return string(t.runes[start:i]), nil
		}
	}

	return "", fmt.Errorf("no closing %s found", string(end))
}

func (t *Template) isNumber(i int) bool {
	return (t.runes[i] >= '0' && t.runes[i] <= '9')
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("need a command")
	}

	cmd, args := os.Args[1], os.Args[2:]
	dicerTmpl := NewTemplate(cmd)
	for _, input := range args {
		diced, err := dicerTmpl.Expand([]string{input})
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", diced)
	}
}
