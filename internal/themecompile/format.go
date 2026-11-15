package themecompile

import (
	"fmt"
	"strings"
)

// maxFormatLength leaves one of the 19 bytes of the format field as the zero byte that ends it.
const maxFormatLength = 18

// formatTokens are the tokens of a datetime format, each with the digit the device knows it by. YYYY comes first so
// that no shorter token is taken where it starts.
var formatTokens = []struct {
	text  string
	digit byte
}{{"YYYY", '1'}, {"MM", '2'}, {"DD", '3'}, {"HH", '4'}, {"mm", '5'}, {"ss", '6'}}

const formatSeparators = ".-/: "

// compileFormat compiles a format: YYYY to 1, MM to 2, DD to 3, HH to 4, mm to 5, ss to 6, and each of the separators
// ". - / : space" to itself. Positions count runes from 1.
func compileFormat(text string) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("want a non-empty format")
	}
	var compiled []byte
	runes := []rune(text)
	for position := 0; position < len(runes); {
		rest := string(runes[position:])
		token := -1
		for index, candidate := range formatTokens {
			if strings.HasPrefix(rest, candidate.text) {
				token = index
				break
			}
		}
		switch {
		case token >= 0:
			compiled = append(compiled, formatTokens[token].digit)
			position += len(formatTokens[token].text)
		case strings.ContainsRune(formatSeparators, runes[position]):
			compiled = append(compiled, byte(runes[position]))
			position++
		default:
			return nil, fmt.Errorf("the character %q at position %d is not a token or a separator", runes[position], position+1)
		}
	}
	if len(compiled) > maxFormatLength {
		return nil, fmt.Errorf("compiles to %d bytes, more than %d", len(compiled), maxFormatLength)
	}
	return compiled, nil
}
