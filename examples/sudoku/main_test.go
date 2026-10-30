package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"
)

const firstSolution = `Solution 1:
2 8 6 1 5 9 7 4 3
4 5 7 6 3 8 2 1 9
3 1 9 7 4 2 5 6 8
8 2 1 9 6 5 4 3 7
6 9 3 8 7 4 1 2 5
7 4 5 3 2 1 8 9 6
5 6 8 2 1 3 9 7 4
1 3 4 5 9 7 6 8 2
9 7 2 4 8 6 3 5 1
`

const emptyBoard = `_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
`

// conflictingBoard repeats the digit 5 in its first row.
const conflictingBoard = `5 5 _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
_ _ _ _ _ _ _ _ _
`

func readBoardFile(t *testing.T) string {
	t.Helper()

	board, err := os.ReadFile("sudoku.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(board)
}

func TestRun(t *testing.T) {
	t.Parallel()

	board := readBoardFile(t)

	tests := []struct {
		name          string
		args          []string
		stdin         string
		wantCode      int
		wantSolutions int
		wantStdout    string
		wantStderr    string
	}{
		{
			name:          "first solution by default",
			stdin:         board,
			wantCode:      exitSolved,
			wantSolutions: 1,
			wantStdout:    firstSolution,
		},
		{
			name:          "limit stops the search",
			args:          []string{"-limit", "2"},
			stdin:         board,
			wantCode:      exitSolved,
			wantSolutions: 2,
		},
		{
			name:          "zero limit finds all solutions",
			args:          []string{"-limit", "0"},
			stdin:         board,
			wantCode:      exitSolved,
			wantSolutions: 8,
		},
		{
			name:          "limit above the number of solutions",
			args:          []string{"-limit", "100"},
			stdin:         board,
			wantCode:      exitSolved,
			wantSolutions: 8,
		},
		{
			name:       "no solution",
			stdin:      conflictingBoard,
			wantCode:   exitNoSolution,
			wantStderr: "sudoku: no solution found\n",
		},
		{
			name:       "invalid board",
			stdin:      "1 2 x",
			wantCode:   exitFailure,
			wantStderr: "sudoku: line 1, column 3: expected digit or underscore symbol\n",
		},
		{
			name:       "empty input",
			stdin:      "",
			wantCode:   exitFailure,
			wantStderr: "sudoku: line 1: unexpected end of input\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			stdin:      board,
			wantCode:   exitFailure,
			wantStderr: "flag provided but not defined: -unknown\nusage: sudoku",
		},
		{
			name:       "negative limit",
			args:       []string{"-limit", "-1"},
			stdin:      board,
			wantCode:   exitFailure,
			wantStderr: `invalid value "-1" for flag -limit`,
		},
		{
			name:       "positional argument",
			args:       []string{"sudoku.txt"},
			stdin:      board,
			wantCode:   exitFailure,
			wantStderr: "sudoku: unexpected argument \"sudoku.txt\"\nusage: sudoku",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantCode:   exitSolved,
			wantStderr: "usage: sudoku [-limit n] < board.txt",
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

func TestRun_ReadFailure(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run(nil, iotest.ErrReader(errors.New("read failure")), &stdout, &stderr)

	if code != exitFailure {
		t.Errorf("exit status %d, want %d", code, exitFailure)
	}
	want := "sudoku: line 1: reading input: read failure\n"
	if stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
	if stdout.Len() != 0 {
		t.Errorf("unexpected stdout:\n%s", stdout.String())
	}
}

func TestRun_WriteFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		stdin string
	}{
		{
			// The solution fits in the output buffer, so the write happens at the flush.
			name:  "single solution",
			stdin: readBoardFile(t),
		},
		{
			// The board has too many solutions to enumerate, so this ends only
			// if the failure stops the search.
			name:  "unbounded search",
			args:  []string{"-limit", "0"},
			stdin: emptyBoard,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), failingWriter{}, &stderr)

			if code != exitFailure {
				t.Errorf("exit status %d, want %d", code, exitFailure)
			}
			want := "sudoku: writing solution: write failure\n"
			if stderr.String() != want {
				t.Errorf("stderr = %q, want %q", stderr.String(), want)
			}
		})
	}
}

func TestPrintSolution_WriteFailure(t *testing.T) {
	t.Parallel()

	solution := strings.Repeat("123456789", 9)
	line := "1 2 3 4 5 6 7 8 9\n"

	tests := []struct {
		name       string
		failAfter  int
		wantOutput string
	}{
		{
			name:      "header",
			failAfter: 0,
		},
		{
			name:       "first line",
			failAfter:  len("Solution 1:\n"),
			wantOutput: "Solution 1:\n",
		},
		{
			name:       "last line",
			failAfter:  len("Solution 1:\n") + 8*len(line),
			wantOutput: "Solution 1:\n" + strings.Repeat(line, 8),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			w := &limitedWriter{w: &output, remaining: tt.failAfter}

			if err := printSolution(w, 1, solution); !errors.Is(err, errWrite) {
				t.Errorf("printSolution() = %v, want %v", err, errWrite)
			}
			if output.String() != tt.wantOutput {
				t.Errorf("output = %q, want %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestPrintSolution(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := printSolution(&output, 7, strings.Repeat("123456789", 9)); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf("Solution 7:\n%s", strings.Repeat("1 2 3 4 5 6 7 8 9\n", 9))
	if output.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", output.String(), want)
	}
}

var errWrite = errors.New("write failure")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}

// limitedWriter fails the first write that does not fit in remaining bytes.
type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if len(p) > l.remaining {
		return 0, errWrite
	}
	l.remaining -= len(p)
	return l.w.Write(p)
}
