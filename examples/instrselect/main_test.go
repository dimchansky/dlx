package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const cheapestProgram = `Expression: load(add(a, 8))
Nodes: 4, rows: 7, tilings: 4

Tiling 1, cost 4:
  ldr R0, [a, #8]            ; ldrimm, cost 4, covers load add 8
  result in R0
`

const allTilings = `Expression: load(add(a, 8))
Nodes: 4, rows: 7, tilings: 4

Tiling 1, cost 4:
  ldr R0, [a, #8]            ; ldrimm, cost 4, covers load add 8
  result in R0

Tiling 2, cost 5:
  add R0, a, #8              ; addi, cost 1, covers add 8
  ldr R1, [R0]               ; ldr, cost 4, covers load
  result in R1

Tiling 3, cost 5:
  mov R0, #8                 ; mov, cost 1, covers 8
  ldr R1, [a, R0]            ; ldrreg, cost 4, covers load add
  result in R1

Tiling 4, cost 6:
  mov R0, #8                 ; mov, cost 1, covers 8
  add R1, a, R0              ; add, cost 1, covers add
  ldr R2, [R1]               ; ldr, cost 4, covers load
  result in R2
`

const arrayElement = `Expression: load(add(a, shl(i, 3)))
Nodes: 6, rows: 9, tilings: 4

Tiling 1, cost 4:
  ldr R0, [a, i, lsl #3]     ; ldrshift, cost 4, covers load add shl 3
  result in R0
`

const multiplyAndAdd = `Expression: add(mul(a, b), c)
Nodes: 5, rows: 6, tilings: 2

Tiling 1, cost 3:
  madd R0, a, b, c           ; madd, cost 3, covers add mul
  result in R0

Tiling 2, cost 4:
  mul R0, a, b               ; mul, cost 3, covers mul
  add R1, R0, c              ; add, cost 1, covers add
  result in R1
`

const oneAdd = `Expression: add(a, b)
Nodes: 3, rows: 3, tilings: 1

Tiling 1, cost 1:
  add R0, a, b               ; add, cost 1, covers add
  result in R0
`

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantCode    int
		wantTilings int
		wantStdout  string
		wantStderr  string
	}{
		{
			name:        "default expression",
			wantTilings: 1,
			wantStdout:  cheapestProgram,
		},
		{
			name:        "limit prints the cheapest few",
			args:        []string{"-limit", "2"},
			wantTilings: 2,
		},
		{
			name:        "zero limit prints all tilings",
			args:        []string{"-limit", "0"},
			wantTilings: 4,
			wantStdout:  allTilings,
		},
		{
			name:        "limit above the number of tilings",
			args:        []string{"-limit", "100"},
			wantTilings: 4,
		},
		{
			name:        "adds is not a second tiling",
			args:        []string{"-expr", "add(a, b)", "-limit", "0"},
			wantTilings: 1,
			wantStdout:  oneAdd,
		},
		{
			name:        "element of an array",
			args:        []string{"-expr", "load(add(a, shl(i, 3)))"},
			wantTilings: 1,
			wantStdout:  arrayElement,
		},
		{
			name:        "two cheapest tilings",
			args:        []string{"-expr", "add(mul(a, b), c)", "-limit", "2"},
			wantTilings: 2,
			wantStdout:  multiplyAndAdd,
		},
		{
			name:       "no tiling",
			args:       []string{"-expr", "shl(a, b)"},
			wantCode:   1,
			wantStderr: "instrselect: no tiling covers every node of shl(a, b)\n",
		},
		{
			name:       "invalid expression",
			args:       []string{"-expr", "sub(a, b)"},
			wantCode:   2,
			wantStderr: "instrselect: column 1: unknown operator \"sub\"\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: instrselect",
		},
		{
			name:       "negative limit",
			args:       []string{"-limit", "-1"},
			wantCode:   2,
			wantStderr: "invalid value \"-1\" for flag -limit",
		},
		{
			name:       "positional argument",
			args:       []string{"add(a, b)"},
			wantCode:   2,
			wantStderr: "instrselect: unexpected argument \"add(a, b)\"\nusage: instrselect",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: instrselect [-expr expression] [-limit n]",
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
			if got := strings.Count(stdout.String(), "Tiling "); got != tt.wantTilings {
				t.Errorf("printed %d tilings, want %d", got, tt.wantTilings)
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
		failOn int
	}{
		{name: "summary", failOn: 1},
		{name: "tiling header", failOn: 2},
		{name: "instruction line", failOn: 3},
		{name: "result line", failOn: 4},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(nil, &failingWriter{failOn: tt.failOn}, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "instrselect: writing output: write failure\n"
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
