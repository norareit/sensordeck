package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// traceLine is one line of the trace, and of the listening commands: the time since the command started in seconds
// with six decimals, padded to 12 characters, a space and the body.
func traceLine(elapsed time.Duration, body string) string {
	return fmt.Sprintf("%12.6f %s", elapsed.Seconds(), body)
}

// reportBytes writes all 64 bytes as two upper-case hexadecimal digits each, separated by single spaces.
func reportBytes(report smartmonitor.Report) string {
	return fmt.Sprintf("% X", report[:])
}

func writeBody(report smartmonitor.Report) string { return "> " + reportBytes(report) }

func readBody(report smartmonitor.Report) string { return "< " + reportBytes(report) }

// tracingTransport writes one line to out for every call on the inner transport, when the call returns. It sees
// reports and not the bytes of the node, so it holds for every transport.
type tracingTransport struct {
	inner   transport
	out     io.Writer
	elapsed func() time.Duration
}

func (tracing *tracingTransport) line(body string) {
	_, _ = fmt.Fprintln(tracing.out, traceLine(tracing.elapsed(), body))
}

func (tracing *tracingTransport) Write(report smartmonitor.Report) error {
	err := tracing.inner.Write(report)
	if err != nil {
		tracing.line(writeBody(report) + " ! " + err.Error())
		return err
	}
	tracing.line(writeBody(report))
	return nil
}

func (tracing *tracingTransport) Read(timeout time.Duration) (smartmonitor.Report, error) {
	report, err := tracing.inner.Read(timeout)
	if err != nil {
		tracing.line("< ! " + err.Error())
		return report, err
	}
	tracing.line(readBody(report))
	return report, nil
}

// Serial passes through and writes no line, because it is no call on the device.
func (tracing *tracingTransport) Serial() string { return tracing.inner.Serial() }

func (tracing *tracingTransport) Reopen() error {
	err := tracing.inner.Reopen()
	if err != nil {
		tracing.line("* reopen ! " + err.Error())
		return err
	}
	tracing.line("* reopen")
	return nil
}

func (tracing *tracingTransport) Close() error { return tracing.inner.Close() }
