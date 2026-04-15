package cli

import "github.com/google/shlex"

func Split(line string) ([]string, error) {
	return shlex.Split(line)
}
