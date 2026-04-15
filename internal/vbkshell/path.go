package vbkshell

import (
	"path"
	"strconv"
	"strings"
)

func normalizePath(p, cwd string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	if p == "" {
		if cwd == "" {
			return "/"
		}
		return cwd
	}

	if len(p) >= 2 && p[1] == ':' {
		p = p[2:]
	}

	if !strings.HasPrefix(p, "/") {
		p = path.Join(cwd, p)
	}

	p = path.Clean(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func humanSize(size uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(size)
	for i, u := range units {
		if v < 1024 || i == len(units)-1 {
			if u == "B" {
				return strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(formatFloat(v, 0), ".0"), ".")) + u
			}
			return formatFloat(v, 1) + u
		}
		v /= 1024
	}
	return "0B"
}

func formatFloat(v float64, decimals int) string {
	if decimals <= 0 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', decimals, 64)
}
