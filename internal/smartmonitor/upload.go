package smartmonitor

import (
	"fmt"
	"time"
)

const (
	reopenFirstPause = time.Second
	reopenRetryPause = 500 * time.Millisecond
	reopenAttempts   = 20
	answerTimeout    = 10 * time.Second
	maxWrites        = 10

	answerACK byte = 0x06
	answerNAK byte = 0x15
)

// Transport is what the upload needs of the outside world.
type Transport interface {
	// Write sends one report.
	Write(report Report) error
	// Read returns one report, or an error once the time limit has passed.
	Read(timeout time.Duration) (Report, error)
	// Reopen lets go of the device as it was opened and tries once to open it again.
	Reopen() error
}

// Stage names the step of the upload at which an error happened.
type Stage int

const (
	StageReset Stage = iota
	StageReopenAfterReset
	StageYmodemCommand
	StageHeader
	StageData
	StageEndOfTransmission
	StageClosing
	StageReopenAfterUpload
)

var stageNames = [...]string{
	StageReset:             "reset",
	StageReopenAfterReset:  "reopening after the reset",
	StageYmodemCommand:     "ymodem command",
	StageHeader:            "header block",
	StageData:              "data block",
	StageEndOfTransmission: "end of transmission",
	StageClosing:           "closing block",
	StageReopenAfterUpload: "reopening after the upload",
}

func (stage Stage) String() string {
	if stage < 0 || int(stage) >= len(stageNames) {
		return fmt.Sprintf("stage %d", int(stage))
	}
	return stageNames[stage]
}

// UploadError tells at which stage the upload failed. Block is the sequence number of the data block counted from 1,
// and 0 at every other stage.
type UploadError struct {
	Stage Stage
	Block int
	Err   error
}

func (uploadError *UploadError) Error() string {
	if uploadError.Stage == StageData {
		return fmt.Sprintf("upload failed at data block %d: %v", uploadError.Block, uploadError.Err)
	}
	return fmt.Sprintf("upload failed at the %s: %v", uploadError.Stage, uploadError.Err)
}

func (uploadError *UploadError) Unwrap() error {
	return uploadError.Err
}

// Upload sends the theme to the device. The pause function is called with the length of every pause, and is how the
// upload waits.
func Upload(transport Transport, theme []byte, pause func(time.Duration)) error {
	steps := []func() *UploadError{
		func() *UploadError { return writeOnly(transport, []Report{ResetCommand()}, StageReset) },
		func() *UploadError { return reopen(transport, pause, StageReopenAfterReset) },
		func() *UploadError {
			return sendItem(transport, []Report{YmodemCommand()}, StageYmodemCommand, 0)
		},
		func() *UploadError {
			return sendItem(transport, toReports(headerBlock(len(theme))), StageHeader, 0)
		},
		func() *UploadError { return sendDataBlocks(transport, theme) },
		func() *UploadError {
			return sendItem(transport, toReports(endOfTransmission()), StageEndOfTransmission, 0)
		},
		func() *UploadError { return writeOnly(transport, toReports(closingBlock()), StageClosing) },
		func() *UploadError { return reopen(transport, pause, StageReopenAfterUpload) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func sendDataBlocks(transport Transport, theme []byte) *UploadError {
	for position, block := range dataBlocks(theme) {
		if err := sendItem(transport, toReports(block), StageData, position+1); err != nil {
			return err
		}
	}
	return nil
}

// sendItem writes the reports and reads one answer, and writes them again on NAK, up to maxWrites writes in all. A
// failed read ends it without writing again, since the device may already have the item.
func sendItem(transport Transport, reports []Report, stage Stage, block int) *UploadError {
	for attempt := 1; attempt <= maxWrites; attempt++ {
		if err := writeReports(transport, reports); err != nil {
			return &UploadError{Stage: stage, Block: block, Err: err}
		}
		answer, err := transport.Read(answerTimeout)
		if err != nil {
			return &UploadError{Stage: stage, Block: block, Err: err}
		}
		switch answer[0] {
		case answerACK:
			return nil
		case answerNAK:
		default:
			return &UploadError{Stage: stage, Block: block, Err: fmt.Errorf("unexpected answer 0x%02x", answer[0])}
		}
	}
	return &UploadError{Stage: stage, Block: block, Err: fmt.Errorf("the device refused the item %d times", maxWrites)}
}

// writeOnly writes the reports and reads nothing, because the device reboots after the closing block.
func writeOnly(transport Transport, reports []Report, stage Stage) *UploadError {
	if err := writeReports(transport, reports); err != nil {
		return &UploadError{Stage: stage, Err: err}
	}
	return nil
}

func writeReports(transport Transport, reports []Report) error {
	for _, report := range reports {
		if err := transport.Write(report); err != nil {
			return err
		}
	}
	return nil
}

// reopen tries to open the device again after it rebooted, whatever kind of error a try gives, because an I/O error
// is normal while the device enumerates.
func reopen(transport Transport, pause func(time.Duration), stage Stage) *UploadError {
	pause(reopenFirstPause)
	var lastErr error
	for attempt := 1; attempt <= reopenAttempts; attempt++ {
		lastErr = transport.Reopen()
		if lastErr == nil {
			return nil
		}
		if attempt < reopenAttempts {
			pause(reopenRetryPause)
		}
	}
	return &UploadError{Stage: stage, Err: fmt.Errorf("the device did not open in %d attempts: %w", reopenAttempts, lastErr)}
}
