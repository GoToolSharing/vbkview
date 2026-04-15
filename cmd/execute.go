package cmd

import "os"

func Execute() int {
	if err := NewRootCmd().Execute(); err != nil {
		_, _ = os.Stderr.WriteString("Error: " + err.Error() + "\n")
		return exitCodeForError(err)
	}
	return exitOK
}
