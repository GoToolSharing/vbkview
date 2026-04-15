package vbkshell

import "testing"

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		in   string
		cwd  string
		want string
	}{
		{"", "/a/b", "/a/b"},
		{"foo", "/a/b", "/a/b/foo"},
		{"..\\bar", "/a/b", "/a/bar"},
		{"C:\\Windows\\System32", "/", "/Windows/System32"},
		{"/etc/../tmp", "/x", "/tmp"},
	}

	for _, tt := range tests {
		got := normalizePath(tt.in, tt.cwd)
		if got != tt.want {
			t.Fatalf("normalizePath(%q,%q)=%q, want %q", tt.in, tt.cwd, got, tt.want)
		}
	}
}
