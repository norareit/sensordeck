package custommetrics

import (
	"fmt"
	"io"
)

// streak tells a metric that gives no value from one that gives none again, so that a log is not filled by a metric
// that stays down. A line says when it is back.
type streak struct{ failing bool }

func (current *streak) observe(stderr io.Writer, name string, result outcome) {
	switch {
	case result.failed() && !current.failing:
		current.failing = true
		_, _ = fmt.Fprintf(stderr, "metric %s: %s\n", name, result.cause)
	case !result.failed() && current.failing:
		current.failing = false
		_, _ = fmt.Fprintf(stderr, "metric %s: read again\n", name)
	}
}
