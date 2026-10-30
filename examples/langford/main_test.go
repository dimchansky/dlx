package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantLines  int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "first arrangement by default",
			wantLines:  1,
			wantStdout: "1 7 1 2 5 6 2 3 4 7 5 3 6 4\n",
		},
		{
			name:       "the only arrangement up to mirrors",
			args:       []string{"-n", "3"},
			wantLines:  1,
			wantStdout: "3 1 2 1 3 2\n",
		},
		{
			name:       "mirrors are separate arrangements",
			args:       []string{"-n", "3", "-mirrors", "-limit", "0"},
			wantLines:  2,
			wantStdout: "3 1 2 1 3 2\n2 3 1 2 1 3\n",
		},
		{
			name:      "limit stops the search",
			args:      []string{"-limit", "3"},
			wantLines: 3,
		},
		{
			name:      "zero limit prints all arrangements",
			args:      []string{"-limit", "0"},
			wantLines: 26,
		},
		{
			name:       "count prints the total",
			args:       []string{"-count"},
			wantLines:  1,
			wantStdout: "Arrangements: 26\n",
		},
		{
			name:       "count with mirrors",
			args:       []string{"-count", "-mirrors"},
			wantLines:  1,
			wantStdout: "Arrangements: 52\n",
		},
		{
			name:       "count ignores limit",
			args:       []string{"-count", "-limit", "1"},
			wantLines:  1,
			wantStdout: "Arrangements: 26\n",
		},
		{
			name:       "no arrangement",
			args:       []string{"-n", "5"},
			wantCode:   1,
			wantStderr: "langford: no arrangement found\n",
		},
		{
			name:       "count with no arrangement",
			args:       []string{"-n", "5", "-count"},
			wantCode:   1,
			wantLines:  1,
			wantStdout: "Arrangements: 0\n",
			wantStderr: "langford: no arrangement found\n",
		},
		{
			name:       "size too small",
			args:       []string{"-n", "0"},
			wantCode:   2,
			wantStderr: "langford: invalid size 0: want 1 to 64\n",
		},
		{
			name:       "size too large",
			args:       []string{"-n", "65"},
			wantCode:   2,
			wantStderr: "langford: invalid size 65: want 1 to 64\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: langford",
		},
		{
			name:       "positional argument",
			args:       []string{"7"},
			wantCode:   2,
			wantStderr: "langford: unexpected argument \"7\"\nusage: langford",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: langford [-n size] [-limit n] [-count] [-mirrors]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(""), &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit status %d, want %d", code, tt.wantCode)
			}
			if got := strings.Count(stdout.String(), "\n"); got != tt.wantLines {
				t.Errorf("printed %d lines, want %d", got, tt.wantLines)
			}
			if tt.wantStdout != "" && stdout.String() != tt.wantStdout {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr:\n%s\nwant it to contain:\n%s", stderr.String(), tt.wantStderr)
			}
			if tt.wantStderr == "" && stderr.Len() != 0 {
				t.Errorf("unexpected stderr:\n%s", stderr.String())
			}
		})
	}
}

func TestRun_WriteFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		failOn int
	}{
		{name: "first line stops the search", args: []string{"-limit", "0"}, failOn: 1},
		{name: "later line stops the search", args: []string{"-limit", "0"}, failOn: 5},
		{name: "total", args: []string{"-count"}, failOn: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			writer := &failingWriter{failOn: tt.failOn}
			code := run(tt.args, strings.NewReader(""), writer, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "langford: writing output: write failure\n"
			if stderr.String() != want {
				t.Errorf("stderr = %q, want %q", stderr.String(), want)
			}
			if writer.writes != tt.failOn {
				t.Errorf("made %d writes, want the search to stop after %d", writer.writes, tt.failOn)
			}
		})
	}
}

// failingWriter accepts writes until write number failOn, which fails.
type failingWriter struct {
	failOn int
	writes int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.failOn {
		return 0, errors.New("write failure")
	}
	return len(p), nil
}
