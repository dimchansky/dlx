package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"testing/iotest"
)

const cheapestRoster = `Rosters covering every flight: 40

Roster 1, cost 3730:
  B1    1150  F101 F102 F103 F104
  B5    1400  F105 F106 F107 F108
  B4    1180  F109 F110 F111 F112
`

func TestRun(t *testing.T) {
	t.Parallel()

	sample := readSample(t)

	tests := []struct {
		name        string
		args        []string
		stdin       string
		wantCode    int
		wantRosters int
		wantStdout  string
		wantStderr  string
	}{
		{
			name:        "cheapest roster by default",
			stdin:       sample,
			wantRosters: 1,
			wantStdout:  cheapestRoster,
		},
		{
			name:        "limit prints the cheapest few",
			args:        []string{"-limit", "3"},
			stdin:       sample,
			wantRosters: 3,
		},
		{
			name:        "zero limit prints all rosters",
			args:        []string{"-limit", "0"},
			stdin:       sample,
			wantRosters: 40,
		},
		{
			name:        "limit above the number of rosters",
			args:        []string{"-limit", "100"},
			stdin:       sample,
			wantRosters: 40,
		},
		{
			name:       "no roster",
			stdin:      "flights F1 F2\nA 1 F1\n",
			wantCode:   1,
			wantStderr: "crew: no roster covers every flight\n",
		},
		{
			name:       "invalid input",
			stdin:      "flights F1\nA 1 F2\n",
			wantCode:   2,
			wantStderr: "crew: line 2: unknown flight \"F2\"\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			stdin:      sample,
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: crew",
		},
		{
			name:       "positional argument",
			args:       []string{"pairings.txt"},
			stdin:      sample,
			wantCode:   2,
			wantStderr: "crew: unexpected argument \"pairings.txt\"\nusage: crew",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: crew [-limit n] < pairings.txt",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit status %d, want %d", code, tt.wantCode)
			}
			if got := strings.Count(stdout.String(), "Roster "); got != tt.wantRosters {
				t.Errorf("printed %d rosters, want %d", got, tt.wantRosters)
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

func TestRun_ReadFailure(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run(nil, iotest.ErrReader(errors.New("read failure")), &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit status %d, want 2", code)
	}
	want := "crew: line 1: reading input: read failure\n"
	if stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
	if stdout.Len() != 0 {
		t.Errorf("unexpected stdout:\n%s", stdout.String())
	}
}

func TestRun_WriteFailure(t *testing.T) {
	t.Parallel()

	sample := readSample(t)

	tests := []struct {
		name   string
		failOn int
	}{
		{name: "summary", failOn: 1},
		{name: "roster header", failOn: 2},
		{name: "pairing line", failOn: 3},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(nil, strings.NewReader(sample), &failingWriter{failOn: tt.failOn}, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "crew: writing output: write failure\n"
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
