package dicer

import (
	"fmt"
	"strconv"
	"strings"
)

// Dicer applies a dicer expression to a given input string. A dicer expression is everything that
// comes after the '%[n' input selector. For example, the template '%[1.3.-$]' has a dicer
// expression '.3.-$]'. The dicer expression is a set of two repeating tokens: the dice character
// and the position selector. The input string is split by the dicer character, with positions
// indexed starting at 1. The selector is, by default, replace (replace the whole string with the
// selected position). To change the selector to remove (remove the selected position), prefix it
// with a '-'. Position is simply an index number (1-based), and also accepts the special character
// '$' to signify the last index. It returns the diced result and any errors encountered.
func Dicer(base string, expr []rune) (string, error) {
	out := base

	// Now we have the start of the dicer expression as `pos`. We have two states: reading a dice character, and
	// reading a dice operation/position. Alternate between both until we hit the end.
	dicerChar := ""
	for pos := 0; pos < len(expr); pos++ {
		char := expr[pos]

		if char == ']' {
			break
		} else if dicerChar == "" {
			dicerChar = string(char)
		} else {
			dicerPosType, dicerPos, dicerPosLength, err := ReadDicerPos(expr[pos:])
			//fmt.Printf("dicerPosType=%v dicerPos=%v dicerPosLength=%v\n", dicerPosType, dicerPos, dicerPosLength)
			if err != nil {
				return "", fmt.Errorf("error trying to read dicerLength: %s", err.Error())
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
			//fmt.Printf("started reading at pos=%d (%s), moving forward %d\n", pos, string(expr[pos]), dicerPosLength-1)
			pos += dicerPosLength - 1

			// Reset state machine
			dicerChar = ""
		}
	}

	return out, nil
}

// Expand a dicer template given a set of strings mapping to %1, %2, etc.
// %n is a shortcut for %[n].
// %[expressions] are expanded by the Dicer.
func Expand(template string, inputs []string) (string, error) {
	runes := []rune(template)
	var out string

	if len(inputs) == 0 {
		return "", fmt.Errorf("at least one input must be specified")
	}

	for pos := 0; pos < len(runes); pos++ {
		char := runes[pos]

		// A '%' dicer escape followed by at least one character means we need to expand it.
		if char == '%' && pos < len(runes)-1 {
			peek := runes[pos+1]
			switch {
			case peek == '%':
				pos++
				out += "%"
				continue
			case peek == '[':
				// Scan ahead to make sure the dicer expression has a closing ']'. We could catch this later
				// parsing char-by-char, but it's a better user experience to see an easy error about a missing
				// ']' instead of seeing that there's an invalid dicer expression (trying to parse chars that
				// were meant to be after the ']').
				end := 0
				for i := pos; i < len(runes); i++ {
					if runes[i] == ']' {
						end = i
						break
					}
				}
				if end == 0 {
					return "", fmt.Errorf("char %d: dicer expression missing closing ]", pos+1)
				}

				dicerExpr := runes[pos+2 : end]
				index, indexLength, err := ReadNumber(dicerExpr)
				if err != nil {
					return "", err
				}

				input, err := inputPick(&inputs, index)
				if err != nil {
					return "", err
				}

				dicerOut, err := Dicer(input, dicerExpr[indexLength:])
				if err != nil {
					return "", err
				}

				pos += len(dicerExpr) + 2 // extra 2 for the []s
				out += dicerOut
				continue
			case isDigit(runes[pos+1]):
				index, indexLength, err := ReadNumber(runes[pos+1:])
				if err != nil {
					return "", err
				}

				input, err := inputPick(&inputs, index)
				if err != nil {
					return "", err
				}

				pos += indexLength
				out += input
				continue
			}
		}

		out += string(char)
	}

	return out, nil
}

// Dicer position constants
const (
	dicerLast   = -1
	dicerRemove = 1
	dicerSelect = 2
)

// ReadDicerPos reads the next full dicer position in a slice of runes.  A dicer position can be:
//    int   type dicerSelect; A positive integer, representing the position of the diced object to select
//    -int  type dicerRemove; A negative integer, representing the position of the diced object to remove
//    $     type dicerSelect; The character '$', representing the last position of the diced
//
// It returns a position type (dicerRemove or dicerSelect), and a position number, the number of
// runes the position took up, and any position parsing error encountered. The position number is
// either a positive integer or the special const dicerLast (-1) which represents the last value.
func ReadDicerPos(runes []rune) (int, int, int, error) {
	pos := 0
	positionType := dicerSelect
	if runes[pos] == '-' {
		positionType = dicerRemove
		pos++
	}

	if isDigit(runes[pos]) {
		dicerPos, dicerPosLength, err := ReadNumber(runes[pos:])

		// what if dicerPos is 0

		if err != nil {
			return 0, 0, 0, err
		}

		return positionType, dicerPos, pos + dicerPosLength, nil
	} else if runes[pos] == '$' {
		return positionType, dicerLast, pos + 1, nil
	}

	return 0, 0, 0, fmt.Errorf("unknown dicer position character %q", string(runes[pos]))
}

// ReadNumber reads the next full number in a slice of runes. A number is specifically a collection
// of [0-9] characters. It returns an int representation of the number read along with the length
// runes that make up the number.
func ReadNumber(runes []rune) (int, int, error) {
	var number string

	for i := 0; i < len(runes) && isDigit(runes[i]); i++ {
		number += string(runes[i])
	}

	if len(number) == 0 { // the rune at start isn't a number!
		return 0, 0, fmt.Errorf("ReadNumber did not find an integer: %v", string(runes[0]))
	}

	i, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return int(i), len(number), nil
}

// isDigit checks if a given rune is a valid looking digit and returns a boolean.
func isDigit(r rune) bool {
	return (r >= '0' && r <= '9')
}

// Pick an input given a dicer index (1-based). Returns the input or an error.
func inputPick(inputs *[]string, index int) (string, error) {
	if index <= 0 {
		return "", fmt.Errorf("index %d: dicer index must be >0", index)
	} else if index > len(*inputs) {
		return "", fmt.Errorf("index %d: out of bounds (inputs size %d)", index, len(*inputs))
	}

	return (*inputs)[index-1], nil
}
