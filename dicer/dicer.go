package dicer

import (
	"fmt"
	"strconv"
	"strings"
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
				dicerOut, dicerExprLength, err := t.dicer(pos+2, inputs)
				if err != nil {
					return "", err
				}

				pos += dicerExprLength + 2 // extra 2 for the []s
				out += dicerOut
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

// Given the starting position of dicer expression (the expr part of %[expr]) and a set of inputs,
// expand the expression's input and dice it up according to the spec.  It returns the diced result,
// the full length of the dicer template (not including the []s or %), and any dice error
// encountered.
func (t *Template) dicer(start int, inputs []string) (string, int, error) {
	// Scan ahead to make sure the dicer expression has a closing ']'. We could catch this later
	// parsing char-by-char, but it's a better user experience to see an easy error about a missing
	// ']' instead of seeing that there's an invalid dicer expression (trying to parse chars that
	// were meant to be after the ']').
	dicerExprLength := 0
	for i := start; i < len(t.runes); i++ {
		if t.runes[i] == ']' {
			dicerExprLength = i - start
			break
		}
	}
	if dicerExprLength == 0 {
		return "", 0, fmt.Errorf("dicer expression: character %d: missing closing ]", start+1)
	}

	// Pull the input position off first
	pos := start
	inputIndex, inputLength, err := t.readNumber(start)
	if err != nil {
		return "", 0, fmt.Errorf("template references invalid input index: %s", err.Error())
	}
	if inputIndex > len(inputs) {
		return "", 0, fmt.Errorf("template references out of bounds input index %d", inputIndex)
	}

	pos += inputLength
	out := inputs[inputIndex-1]

	// Now we have the dicer expression in `expr`. We have two states: reading a dice character, and
	// reading a dice position. Alternate between both until we hit the end.
	dicerChar := ""
	dicerPos := 0
	for ; pos < len(t.runes); pos++ {
		char := t.runes[pos]

		if char == ']' {
			break
		} else if dicerChar == "" {
			dicerChar = string(char)
		} else {
			// TODO: readNumber isn't right, we need readDicerPos. It can be a number, a negative
			// number, or a $.
			var dicerPosLength int
			dicerPos, dicerPosLength, err = t.readNumber(pos)
			if err != nil {
				return "", 0, fmt.Errorf("error trying to read dicerLength: %s", err.Error())
			}
			outParts := strings.Split(out, dicerChar)

			// If they ask for a position we don't have, the output becomes empty and we can
			// short-cicuit.
			if dicerPos > len(outParts) {
				out = ""
				break
			}

			out = outParts[dicerPos-1]

			// The for loop will advance by 1 (pos++), advance any extra chars here if the position
			// is more than one character (e.g. '%[1/10]')
			pos += dicerPosLength - 1

			// Reset state machine
			dicerChar = ""
		}
	}

	return out, dicerExprLength, nil
}

// Given a slice of runes with the first element is a [0-9], read runes as long as we get [0-9]s and
// return an int representation of these numbers along with the number of runes we consumed.
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

func (t *Template) isNumber(i int) bool {
	return (t.runes[i] >= '0' && t.runes[i] <= '9')
}
