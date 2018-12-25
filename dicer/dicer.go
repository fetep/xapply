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

	// Now we have the start of the dicer expression as `pos`. We have two states: reading a dice character, and
	// reading a dice operation/position. Alternate between both until we hit the end.
	dicerChar := ""
	for ; pos < len(t.runes); pos++ {
		char := t.runes[pos]

		if char == ']' {
			break
		} else if dicerChar == "" {
			dicerChar = string(char)
		} else {
			dicerPosType, dicerPos, dicerPosLength, err := t.readDicerPos(pos)
			if err != nil {
				return "", 0, fmt.Errorf("error trying to read dicerLength: %s", err.Error())
			}

			outParts := strings.Split(out, dicerChar)

			// Translate the special token dicerLast to the dicer position of the last part
			if dicerPos == dicerLast {
				dicerPos = len(outParts)
			}

			// 1-based dicer position to 0-based array index
			dicerPosIndex := dicerPos - 1

			switch dicerPosType {
			case dicerSelect:
				// If they ask for a position we don't have in select mode, the output becomes empty and we can
				// short-cicuit.
				if dicerPos > len(outParts) {
					out = ""
					break
				}

				out = outParts[dicerPos-1]
			case dicerRemove:
				// If they ask for a position we don't have in remove mode, simply do nothing. Only
				// handle the case where we have an element to remove.
				if dicerPos <= len(outParts) {
					outParts = append(outParts[:dicerPosIndex], outParts[dicerPosIndex+1:]...)
				}

				out = strings.Join(outParts, dicerChar)
			default:
				panic(fmt.Errorf("unknown dicerPosType: %d", dicerPosType))
			}

			// The for loop will advance by 1 (pos++), advance any extra chars here if the position
			// is more than one character (e.g. '%[1/10]')
			pos += dicerPosLength - 1

			// Reset state machine
			dicerChar = ""
		}
	}

	return out, dicerExprLength, nil
}

// Dicer position constants
const (
	dicerLast   = -1
	dicerRemove = 1
	dicerSelect = 2
)

// Given a slice of runes, read runes as long as we get valid looking dicer position. A dicer
// position can be:
//    int   type dicerSelect; A positive integer, representing the position of the diced object to select
//    -int  type dicerRemove; A negative integer, representing the position of the diced object to remove
//    $     type dicerSelectLast; The character '$', representing the last position of the diced
//
// It returns a position type (dicerRemove or dicerSelect), and a position number, the number of
// runes the position took up, and any position parsing error encountered. The position number is
// either a positive integer or the special const dicerLast which represents the last value.
func (t *Template) readDicerPos(start int) (int, int, int, error) {
	pos := start
	positionType := dicerSelect
	if t.runes[pos] == '-' {
		positionType = dicerRemove
		pos++
	}

	switch {
	case t.isNumber(pos):
		dicerPos, dicerPosLength, err := t.readNumber(pos)

		// what if dicerPos is 0

		if err != nil {
			return 0, 0, 0, err
		}

		return positionType, dicerPos, (pos - start) + dicerPosLength, nil
	case t.runes[pos] == '$':
		return positionType, dicerLast, (pos - start) + 1, nil
	}

	return 0, 0, 0, fmt.Errorf("unknown dicer position character %q", string(t.runes[pos]))
}

// Given a slice of runes with the first element is a [0-9], read runes as long as we get [0-9]s and
// return an int representation of these numbers along with the length runes that make up the number.
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
