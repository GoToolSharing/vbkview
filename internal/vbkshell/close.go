package vbkshell

import (
	"fmt"
	"io"
	"os"
)

func closeWithWarning(name string, closer io.Closer) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close %s: %v\n", name, err)
	}
}
