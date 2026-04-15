package main

import "os"

func main() {
	if err := NewRootCmd().Execute(); err != nil {
		_, _ = os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(exitCodeForError(err))
	}
}
