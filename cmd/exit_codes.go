package cmd

import (
	"errors"
	"strings"

	vbk "github.com/GoToolSharing/vbktoolkit"
)

const (
	exitOK               = 0
	exitGenericError     = 1
	exitUsageError       = 2
	exitNotFound         = 3
	exitNotDirectory     = 4
	exitIsDirectory      = 5
	exitUnsupportedData  = 6
	exitInvalidInputData = 7
)

func exitCodeForError(err error) int {
	if err == nil {
		return exitOK
	}

	if errors.Is(err, vbk.ErrFileNotFound) {
		return exitNotFound
	}
	if errors.Is(err, vbk.ErrNotDirectory) {
		return exitNotDirectory
	}
	if errors.Is(err, vbk.ErrIsDirectory) {
		return exitIsDirectory
	}
	if errors.Is(err, vbk.ErrUnsupportedBlock) || errors.Is(err, vbk.ErrUnsupportedCompress) || errors.Is(err, vbk.ErrUnsupportedProperty) {
		return exitUnsupportedData
	}
	if errors.Is(err, vbk.ErrNoActiveSlot) || errors.Is(err, vbk.ErrVBK) || errors.Is(err, vbk.ErrIndexOutOfRange) {
		return exitInvalidInputData
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unknown command") || strings.Contains(msg, "required") || strings.Contains(msg, "invalid argument") {
		return exitUsageError
	}

	return exitGenericError
}
