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
			name:      "all systems by default",
			wantLines: 26,
		},
		{
			name:      "default limit stops the printing",
			args:      []string{"-n", "12"},
			wantLines: 30,
		},
		{
			name:       "the whole list for a small number",
			args:       []string{"-n", "4"},
			wantLines:  5,
			wantStdout: "0 mod 1\n0 mod 2, 1 mod 2\n0 mod 2, 1 mod 4, 3 mod 4\n1 mod 2, 0 mod 4, 2 mod 4\n0 mod 4, 1 mod 4, 2 mod 4, 3 mod 4\n",
		},
		{
			name:       "limit stops the search",
			args:       []string{"-limit", "1"},
			wantLines:  1,
			wantStdout: "0 mod 1\n",
		},
		{
			name:      "zero limit prints all systems",
			args:      []string{"-n", "12", "-limit", "0"},
			wantLines: 206,
		},
		{
			name:       "count prints the total",
			args:       []string{"-n", "16", "-count"},
			wantLines:  1,
			wantStdout: "Systems: 677\n",
		},
		{
			name:       "count ignores limit",
			args:       []string{"-count", "-limit", "1"},
			wantLines:  1,
			wantStdout: "Systems: 26\n",
		},
		{
			name:       "the smallest number",
			args:       []string{"-n", "1"},
			wantLines:  1,
			wantStdout: "0 mod 1\n",
		},
		{
			name:       "distinct finds nothing",
			args:       []string{"-n", "12", "-distinct"},
			wantCode:   1,
			wantStderr: "coveringsystems: no system found\n",
		},
		{
			name:       "distinct finds nothing for the smallest number",
			args:       []string{"-n", "1", "-distinct"},
			wantCode:   1,
			wantStderr: "coveringsystems: no system found\n",
		},
		{
			name:       "count with no system",
			args:       []string{"-distinct", "-count"},
			wantCode:   1,
			wantLines:  1,
			wantStdout: "Systems: 0\n",
			wantStderr: "coveringsystems: no system found\n",
		},
		{
			name:       "number too small",
			args:       []string{"-n", "0"},
			wantCode:   2,
			wantStderr: "coveringsystems: invalid number 0: want 1 to 4096\n",
		},
		{
			name:       "number too large",
			args:       []string{"-n", "4097"},
			wantCode:   2,
			wantStderr: "coveringsystems: invalid number 4097: want 1 to 4096\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: coveringsystems",
		},
		{
			name:       "positional argument",
			args:       []string{"8"},
			wantCode:   2,
			wantStderr: "coveringsystems: unexpected argument \"8\"\nusage: coveringsystems",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: coveringsystems [-n number] [-limit n] [-count] [-distinct]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)

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
			code := run(tt.args, writer, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "coveringsystems: writing output: write failure\n"
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
