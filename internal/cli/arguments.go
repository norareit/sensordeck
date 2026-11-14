package cli

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

func invalidPair(argument string) error {
	return fmt.Errorf(`invalid argument %q: want <tag>=<value>`, argument)
}

// parseValues reads the arguments <tag>=<value> of the command values.
func parseValues(args []string) ([]smartmonitor.Value, error) {
	if len(args) == 0 {
		return nil, errors.New("values needs at least one <tag>=<value>")
	}
	values := make([]smartmonitor.Value, 0, len(args))
	for _, argument := range args {
		tagText, valueText, found := strings.Cut(argument, "=")
		if !found {
			return nil, invalidPair(argument)
		}
		tag, tagIsNumber := parseNumber(tagText)
		value, valueIsNumber := parseNumber(valueText)
		if !tagIsNumber || !valueIsNumber {
			return nil, invalidPair(argument)
		}
		if tag > 255 {
			return nil, fmt.Errorf(`invalid argument %q: the tag is outside 0 to 255`, argument)
		}
		if value > 65535 {
			return nil, fmt.Errorf(`invalid argument %q: the value is outside 0 to 65535`, argument)
		}
		values = append(values, smartmonitor.Value{Tag: uint8(tag), Value: uint16(value)})
	}
	return values, nil
}

// parseNumber reads decimal digits, or 0x or 0X and hexadecimal digits, and nothing else. Leading zeros stay decimal,
// which Go's own prefix rules would read as octal.
func parseNumber(text string) (uint64, bool) {
	digits, base := text, 10
	if hexadecimal, found := strings.CutPrefix(text, "0x"); found {
		digits, base = hexadecimal, 16
	} else if hexadecimal, found := strings.CutPrefix(text, "0X"); found {
		digits, base = hexadecimal, 16
	}
	if digits == "" || !allDigits(digits, base) {
		return 0, false
	}
	number, err := strconv.ParseUint(digits, base, 64)
	return number, err == nil
}

// allDigits rejects the signs, spaces and underscores that strconv would otherwise tolerate or reject unevenly.
func allDigits(text string, base int) bool {
	for _, character := range text {
		isDecimal := character >= '0' && character <= '9'
		isHexadecimal := base == 16 && (character >= 'a' && character <= 'f' || character >= 'A' && character <= 'F')
		if !isDecimal && !isHexadecimal {
			return false
		}
	}
	return true
}

// parseRawBytes joins the hexadecimal arguments of the command raw into one report, filled up with 0x00.
func parseRawBytes(args []string) (smartmonitor.Report, error) {
	var report smartmonitor.Report
	if len(args) == 0 {
		return report, errors.New("raw needs at least one byte")
	}
	var data []byte
	for _, argument := range args {
		decoded, err := hex.DecodeString(argument)
		if err != nil || len(decoded) == 0 {
			return report, fmt.Errorf(`invalid argument %q: want pairs of hexadecimal digits`, argument)
		}
		data = append(data, decoded...)
	}
	if len(data) > len(report) {
		return report, fmt.Errorf("raw takes at most %d bytes, got %d", len(report), len(data))
	}
	copy(report[:], data)
	return report, nil
}

// parseSeconds reads the value of --seconds: decimal digits only, 1 or more.
func parseSeconds(text string) (int, error) {
	seconds, isNumber := parseNumber(text)
	if !isNumber || strings.HasPrefix(strings.ToLower(text), "0x") || seconds < 1 || seconds > 1<<31 {
		return 0, fmt.Errorf(`invalid --seconds %q: want a whole number of 1 or more`, text)
	}
	return int(seconds), nil
}
