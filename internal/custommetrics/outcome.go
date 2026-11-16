package custommetrics

import (
	"errors"
	"fmt"
	"io/fs"
	"math"

	"github.com/norareit/sensordeck/internal/config"
)

// outcome is what one reading of a metric gave: a value, or, when the cause is not empty, the reason it has none.
type outcome struct {
	value float64
	cause string
}

func (result outcome) failed() bool { return result.cause != "" }

// fromText takes the number of a text read from source, as the value of the definition.
func fromText(text, source string, definition config.Metric) outcome {
	number, found := Number(text)
	if !found {
		return outcome{cause: "no number in " + source}
	}
	return scaled(outcome{value: number}, definition)
}

// scaled applies the scale and the offset of the definition to the number of a successful outcome.
func scaled(result outcome, definition config.Metric) outcome {
	if result.failed() {
		return result
	}
	value := Value(result.value, definition)
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return outcome{cause: "the number is not finite after the scale and the offset"}
	}
	return outcome{value: value}
}

func unreadable(path string, err error) outcome {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		err = pathError.Err
	}
	return outcome{cause: fmt.Sprintf("cannot read %s: %v", path, err)}
}
