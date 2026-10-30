package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const firstC60Structure = "0-1 2-3 4-25 5-6 7-8 9-26 10-11 12-35 13-14 15-16 17-40 18-19 20-21 22-45 " +
	"23-24 27-50 28-29 30-31 32-39 33-34 36-37 38-56 41-42 43-44 46-47 48-49 51-52 53-54 55-59 57-58\n"

const naphthaleneClar = `Sextets  Covers   Sets Cannot grow
      0       3      1           0
      1       2      2           2
  Total       5      3           2
sextets 1, double bonds 0-1 2-3
sextets 0, double bonds 5-6 7-8
`

const benzeneClar = `Sextets  Covers   Sets Cannot grow
      0       2      1           0
      1       1      1           1
  Total       3      2           1
sextets 0, double bonds none
`

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantLines  int
		wantStdout string
		wantStderr string
		slow       bool
	}{
		{name: "first structure by default", wantLines: 1, wantStdout: firstC60Structure},
		{
			name:       "molecule flag",
			args:       []string{"-molecule", "benzene", "-limit", "0"},
			wantLines:  2,
			wantStdout: "0-1 2-3 4-5\n0-5 1-2 3-4\n",
		},
		{
			name:       "zero limit prints all structures",
			args:       []string{"-molecule", "naphthalene", "-limit", "0"},
			wantLines:  3,
			wantStdout: "0-1 2-3 4-5 6-7 8-9\n0-1 2-3 4-9 5-6 7-8\n0-9 1-2 3-4 5-6 7-8\n",
		},
		{name: "limit stops the search", args: []string{"-limit", "7"}, wantLines: 7},
		{
			name:       "count prints the total",
			args:       []string{"-count"},
			wantLines:  1,
			wantStdout: "Kekule structures: 12500\n",
			slow:       true,
		},
		{
			name:       "count ignores limit",
			args:       []string{"-molecule", "naphthalene", "-count", "-limit", "1"},
			wantLines:  1,
			wantStdout: "Kekule structures: 3\n",
		},
		{
			name:       "clar table",
			args:       []string{"-molecule", "naphthalene", "-clar"},
			wantLines:  6,
			wantStdout: naphthaleneClar,
		},
		{
			name:       "clar with no double bonds left",
			args:       []string{"-molecule", "benzene", "-clar"},
			wantLines:  5,
			wantStdout: benzeneClar,
		},
		{
			name:       "no structure",
			args:       []string{"-molecule", "benzyl"},
			wantCode:   1,
			wantStderr: "fullerene: no Kekule structure found\n",
		},
		{
			name:       "count with no structure",
			args:       []string{"-molecule", "benzyl", "-count"},
			wantCode:   1,
			wantLines:  1,
			wantStdout: "Kekule structures: 0\n",
			wantStderr: "fullerene: no Kekule structure found\n",
		},
		{
			name:       "clar with no structure",
			args:       []string{"-molecule", "benzyl", "-clar"},
			wantCode:   1,
			wantStderr: "fullerene: no Kekule structure found\n",
		},
		{
			name:       "unknown molecule",
			args:       []string{"-molecule", "graphite"},
			wantCode:   2,
			wantStderr: "fullerene: unknown molecule \"graphite\": want one of c60, benzene, naphthalene, benzyl\n",
		},
		{
			name:       "count and clar together",
			args:       []string{"-count", "-clar"},
			wantCode:   2,
			wantStderr: "fullerene: -count and -clar cannot be combined\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow with -race")
			}

			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("code = %d, want %d", code, tt.wantCode)
			}
			if got := strings.Count(stdout.String(), "\n"); got != tt.wantLines {
				t.Errorf("got %d lines, want %d\n%s", got, tt.wantLines, stdout.String())
			}
			if tt.wantStdout != "" && stdout.String() != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
			if stderr.String() != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestRunUsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStderr string
	}{
		{name: "extra argument", args: []string{"extra"}, wantCode: exitFailure, wantStderr: "unexpected argument"},
		{name: "bad flag", args: []string{"-nope"}, wantCode: exitFailure, wantStderr: "usage: fullerene"},
		{name: "bad limit", args: []string{"-limit", "-1"}, wantCode: exitFailure, wantStderr: "usage: fullerene"},
		{name: "help", args: []string{"-help"}, wantCode: exitOK, wantStderr: "exit status: 0 found"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			if code := run(tt.args, &stdout, &stderr); code != tt.wantCode {
				t.Errorf("code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want nothing", stdout.String())
			}
		})
	}
}

// failingWriter fails every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestRunWriteFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "structures", args: []string{"-limit", "0"}},
		{name: "count", args: []string{"-molecule", "benzene", "-count"}},
		{name: "clar", args: []string{"-molecule", "benzene", "-clar"}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			if code := run(tt.args, failingWriter{}, &stderr); code != exitFailure {
				t.Errorf("code = %d, want %d", code, exitFailure)
			}
			if want := "fullerene: writing output: disk full\n"; stderr.String() != want {
				t.Errorf("stderr = %q, want %q", stderr.String(), want)
			}
		})
	}
}
