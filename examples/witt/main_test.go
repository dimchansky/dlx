package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const fanoPlane = `Solution 1:
0 1 2
0 3 4
0 5 6
1 3 5
1 4 6
2 3 6
2 4 5
`

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		wantCode      int
		wantSolutions int
		wantLines     int
		wantStdout    string
		wantStderr    string
	}{
		{
			name:          "Witt design by default",
			wantSolutions: 1,
			wantLines:     133,
		},
		{
			name:          "Fano plane",
			args:          []string{"-t", "2", "-k", "3", "-v", "7"},
			wantSolutions: 1,
			wantStdout:    fanoPlane,
		},
		{
			name:          "limit stops the search",
			args:          []string{"-limit", "3"},
			wantSolutions: 3,
			wantLines:     399,
		},
		{
			name:          "zero limit prints all solutions",
			args:          []string{"-t", "2", "-k", "3", "-v", "7", "-limit", "0"},
			wantSolutions: 30,
			wantLines:     240,
		},
		{
			name:       "count prints the total",
			args:       []string{"-t", "2", "-k", "3", "-v", "7", "-count"},
			wantStdout: "Solutions: 30\n",
		},
		{
			name:       "count ignores limit",
			args:       []string{"-t", "2", "-k", "3", "-v", "7", "-count", "-limit", "1"},
			wantStdout: "Solutions: 30\n",
		},
		{
			name:       "no solution",
			args:       []string{"-t", "2", "-k", "3", "-v", "8"},
			wantCode:   1,
			wantStderr: "witt: no solution found\n",
		},
		{
			name:       "count with no solution",
			args:       []string{"-t", "2", "-k", "3", "-v", "8", "-count"},
			wantCode:   1,
			wantStdout: "Solutions: 0\n",
			wantStderr: "witt: no solution found\n",
		},
		{
			name:       "t is zero",
			args:       []string{"-t", "0"},
			wantCode:   2,
			wantStderr: "witt: invalid parameters t=0 k=6 v=12: want 1 <= t <= k <= v <= 60\n",
		},
		{
			name:       "t above k",
			args:       []string{"-t", "7"},
			wantCode:   2,
			wantStderr: "witt: invalid parameters t=7 k=6 v=12: want 1 <= t <= k <= v <= 60\n",
		},
		{
			name:       "k above v",
			args:       []string{"-k", "13"},
			wantCode:   2,
			wantStderr: "witt: invalid parameters t=5 k=13 v=12: want 1 <= t <= k <= v <= 60\n",
		},
		{
			name:       "too many points",
			args:       []string{"-v", "61"},
			wantCode:   2,
			wantStderr: "witt: invalid parameters t=5 k=6 v=61: want 1 <= t <= k <= v <= 60\n",
		},
		{
			name:       "matrix too large",
			args:       []string{"-t", "2", "-k", "8", "-v", "57"},
			wantCode:   2,
			wantStderr: "witt: S(2,8,57) needs 1652411475 rows with 28 ones each: the limit is 20000000 ones in all\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: witt",
		},
		{
			name:       "positional argument",
			args:       []string{"12"},
			wantCode:   2,
			wantStderr: "witt: unexpected argument \"12\"\nusage: witt",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: witt [-t n] [-k n] [-v n] [-limit n] [-count]",
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
			if got := strings.Count(stdout.String(), "Solution "); got != tt.wantSolutions {
				t.Errorf("printed %d solutions, want %d", got, tt.wantSolutions)
			}
			if got := strings.Count(stdout.String(), "\n"); tt.wantLines != 0 && got != tt.wantLines {
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
		name string
		args []string
	}{
		{name: "system stops the search", args: []string{"-limit", "0"}},
		{name: "total", args: []string{"-t", "2", "-k", "3", "-v", "7", "-count"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(tt.args, failingWriter{}, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "witt: writing output: write failure\n"
			if stderr.String() != want {
				t.Errorf("stderr = %q, want %q", stderr.String(), want)
			}
		})
	}
}

// failingWriter fails every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failure")
}
