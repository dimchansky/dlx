package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const firstSolution = `Solution 1:
Q . . . . . . .
. . . . Q . . .
. . . . . . . Q
. . . . . Q . .
. . Q . . . . .
. . . . . . Q .
. Q . . . . . .
. . . Q . . . .
`

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		wantCode      int
		wantSolutions int
		wantStdout    string
		wantStderr    string
	}{
		{
			name:          "first solution by default",
			wantSolutions: 1,
			wantStdout:    firstSolution,
		},
		{
			name:          "limit stops the search",
			args:          []string{"-limit", "3"},
			wantSolutions: 3,
		},
		{
			name:          "zero limit prints all solutions",
			args:          []string{"-n", "6", "-limit", "0"},
			wantSolutions: 4,
		},
		{
			name:          "smallest board",
			args:          []string{"-n", "1"},
			wantSolutions: 1,
			wantStdout:    "Solution 1:\nQ\n",
		},
		{
			name:       "count prints the total",
			args:       []string{"-count"},
			wantStdout: "Solutions: 92\n",
		},
		{
			name:       "count ignores limit",
			args:       []string{"-count", "-limit", "1"},
			wantStdout: "Solutions: 92\n",
		},
		{
			name:       "no solution",
			args:       []string{"-n", "3"},
			wantCode:   1,
			wantStderr: "queens: no solution found\n",
		},
		{
			name:       "count with no solution",
			args:       []string{"-n", "2", "-count"},
			wantCode:   1,
			wantStdout: "Solutions: 0\n",
			wantStderr: "queens: no solution found\n",
		},
		{
			name:       "size too small",
			args:       []string{"-n", "0"},
			wantCode:   2,
			wantStderr: "queens: invalid board size 0: want 1 to 64\n",
		},
		{
			name:       "size too large",
			args:       []string{"-n", "65"},
			wantCode:   2,
			wantStderr: "queens: invalid board size 65: want 1 to 64\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: queens",
		},
		{
			name:       "positional argument",
			args:       []string{"8"},
			wantCode:   2,
			wantStderr: "queens: unexpected argument \"8\"\nusage: queens",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: queens [-n size] [-limit n] [-count]",
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
			if got := strings.Count(stdout.String(), "Solution "); got != tt.wantSolutions {
				t.Errorf("printed %d solutions, want %d", got, tt.wantSolutions)
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
		{name: "header stops the search", args: []string{"-limit", "0"}, failOn: 1},
		{name: "middle of the board", failOn: 3},
		{name: "total", args: []string{"-count"}, failOn: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(""), &failingWriter{failOn: tt.failOn}, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "queens: writing output: write failure\n"
			if stderr.String() != want {
				t.Errorf("stderr = %q, want %q", stderr.String(), want)
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
