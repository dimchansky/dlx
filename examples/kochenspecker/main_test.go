package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const noColouring = `Set peres33: 33 rays in 3 dimensions
Bases: 16
Pairs: 24
Matrix: 40 columns, 57 rows
Colourings: 0
`

const withoutRayOne = `Set peres33: 33 rays in 3 dimensions
Removed: ray 1 (1, 0, 0)
Bases: 12
Pairs: 28
Matrix: 40 columns, 60 rows
Colouring 1: green rays 2 4 8 12 14 16 19 23 27 30
Colourings: 152
`

const twoWithoutRayOne = `Set peres33: 33 rays in 3 dimensions
Removed: ray 1 (1, 0, 0)
Bases: 12
Pairs: 28
Matrix: 40 columns, 60 rows
Colouring 1: green rays 2 4 8 12 14 16 19 23 27 30
Colouring 2: green rays 2 4 8 12 14 16 19 23 27
Colourings: 152
`

const basesOnly = `Set peres33: 33 rays in 3 dimensions
Bases: 16
Pairs: 0
Matrix: 16 columns, 33 rows
Colourings: 3072
`

const usageLine = "usage: kochenspecker"

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string // exact unless wantPart is set
		wantPart   string // part of stdout
		wantStderr string
	}{
		{
			name:       "the 33 rays have no colouring",
			wantCode:   1,
			wantStdout: noColouring,
			wantStderr: "kochenspecker: no solution found\n",
		},
		{
			name:       "the 24 rays have no colouring",
			args:       []string{"-set", "peres24"},
			wantCode:   1,
			wantPart:   "Colourings: 0\n",
			wantStderr: "kochenspecker: no solution found\n",
		},
		{
			name:       "the 18 rays have no colouring",
			args:       []string{"-set", "cabello18", "-bases-only"},
			wantCode:   1,
			wantPart:   "Matrix: 9 columns, 18 rows\nColourings: 0\n",
			wantStderr: "kochenspecker: no solution found\n",
		},
		{
			name:       "removing a ray allows colourings",
			args:       []string{"-remove", "1"},
			wantStdout: withoutRayOne,
		},
		{
			name:       "the second colouring is the one in the paper",
			args:       []string{"-remove", "1", "-limit", "2"},
			wantStdout: twoWithoutRayOne,
		},
		{
			name:     "limit prints more colourings",
			args:     []string{"-remove", "4", "-limit", "2"},
			wantPart: "Colouring 2: green rays",
		},
		{
			name:     "zero limit prints all colourings",
			args:     []string{"-remove", "4", "-limit", "0"},
			wantPart: "Colouring 16: green rays",
		},
		{
			name:       "bases only prints the total",
			args:       []string{"-bases-only"},
			wantStdout: basesOnly,
		},
		{
			name:     "list the rays",
			args:     []string{"-rays"},
			wantPart: "Rays:\n1 (1, 0, 0)\n",
		},
		{
			name:     "list the groups",
			args:     []string{"-rays"},
			wantPart: "Bases:\n1 2 3\n1 4 5\n",
		},
		{
			name:     "list leaves out the removed ray",
			args:     []string{"-rays", "-remove", "1", "-set", "peres24"},
			wantPart: "Rays:\n2 (0, 1, 0, 0)\n",
		},
		{
			name:     "critical set",
			args:     []string{"-critical", "-set", "cabello18"},
			wantPart: "Without ray 4 (1, -1, 0, 0): 12 colourings\n",
		},
		{
			name:     "critical total",
			args:     []string{"-critical", "-set", "cabello18"},
			wantPart: "Colourable after removing one ray: 18 of 18\n",
		},
		{
			name:       "set that is not critical",
			args:       []string{"-critical", "-set", "peres24"},
			wantCode:   1,
			wantPart:   "Colourable after removing one ray: 0 of 24\n",
			wantStderr: "kochenspecker: no solution found\n",
		},
		{
			name:       "unknown set",
			args:       []string{"-set", "peres99"},
			wantCode:   2,
			wantStderr: "kochenspecker: unknown set \"peres99\": want peres33, peres24, cabello18\n",
		},
		{
			name:       "ray too large",
			args:       []string{"-remove", "34"},
			wantCode:   2,
			wantStderr: "kochenspecker: invalid ray 34: want 1 to 33, or 0 for none\n",
		},
		{
			name:       "negative ray",
			args:       []string{"-remove", "-1"},
			wantCode:   2,
			wantStderr: "kochenspecker: invalid ray -1: want 1 to 33, or 0 for none\n",
		},
		{
			name:       "critical with remove",
			args:       []string{"-critical", "-remove", "1"},
			wantCode:   2,
			wantStderr: "kochenspecker: -critical cannot be combined",
		},
		{
			name:       "critical with rays",
			args:       []string{"-critical", "-rays"},
			wantCode:   2,
			wantStderr: "kochenspecker: -critical cannot be combined",
		},
		{
			name:       "bases only with remove",
			args:       []string{"-bases-only", "-remove", "1"},
			wantCode:   2,
			wantStderr: "kochenspecker: -bases-only cannot be combined with -remove\n",
		},
		{
			name:       "limit with bases only",
			args:       []string{"-limit", "5", "-bases-only"},
			wantCode:   2,
			wantStderr: "kochenspecker: -limit cannot be combined with -bases-only, -rays or -critical\n",
		},
		{
			name:       "limit with rays",
			args:       []string{"-rays", "-limit", "5"},
			wantCode:   2,
			wantStderr: "kochenspecker: -limit cannot be combined",
		},
		{
			name:       "limit with critical",
			args:       []string{"-critical", "-limit", "1"},
			wantCode:   2,
			wantStderr: "kochenspecker: -limit cannot be combined",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\n" + usageLine,
		},
		{
			name:       "positional argument",
			args:       []string{"peres33"},
			wantCode:   2,
			wantStderr: "kochenspecker: unexpected argument \"peres33\"\n" + usageLine,
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: usageLine,
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
			if tt.wantStdout != "" && stdout.String() != tt.wantStdout {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stdout.String(), tt.wantPart) {
				t.Errorf("stdout:\n%s\nwant it to contain:\n%s", stdout.String(), tt.wantPart)
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
		{name: "summary", failOn: 1},
		{name: "colouring stops the search", args: []string{"-remove", "1", "-limit", "0"}, failOn: 2},
		{name: "total", failOn: 2},
		{name: "ray list", args: []string{"-rays"}, failOn: 1},
		{name: "first removal", args: []string{"-critical", "-set", "cabello18"}, failOn: 1},
		{name: "critical total", args: []string{"-critical", "-set", "cabello18"}, failOn: 19},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(tt.args, &failingWriter{failOn: tt.failOn}, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "kochenspecker: writing output: write failure\n"
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
