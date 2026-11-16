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
		value, valueIsNumber, valueInRange := parseValue(valueText)
		if !tagIsNumber || !valueIsNumber {
			return nil, invalidPair(argument)
		}
		if tag > 255 {
			return nil, fmt.Errorf(`invalid argument %q: the tag is outside 0 to 255`, argument)
		}
		if !valueInRange {
			return nil, fmt.Errorf(`invalid argument %q: the value is outside -32768 to 65535`, argument)
		}
		values = append(values, smartmonitor.Value{Tag: uint8(tag), Value: value})
	}
	return values, nil
}

// parseValue reads a number with an optional leading "-". A value from 0 to 65535 is the 16-bit pattern that goes on
// the wire, so 65535 and -1 give the same value, and a negative one is its two's complement.
func parseValue(text string) (value int16, isNumber, inRange bool) {
	magnitudeText, negative := strings.CutPrefix(text, "-")
	magnitude, isNumber := parseNumber(magnitudeText)
	if !isNumber {
		return 0, false, false
	}
	if negative {
		return int16(-int64(min(magnitude, 32769))), true, magnitude <= 32768
	}
	return int16(uint16(min(magnitude, 65536))), true, magnitude <= 65535
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
	// A number too large for 64 bits is still a number. ParseUint gives the largest value then, which every range
	// check refuses with the error of its field.
	return number, err == nil || errors.Is(err, strconv.ErrRange)
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
