package main

import (
	"bytes"
	"errors"
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
			wantCode:      0,
			wantSolutions: 1,
			wantStdout:    firstSolution,
		},
		{
			name:          "limit stops the search",
			args:          []string{"-limit", "2"},
			stdin:         board,
			wantCode:      0,
			wantSolutions: 2,
		},
		{
			name:          "zero limit finds all solutions",
			args:          []string{"-limit", "0"},
			stdin:         board,
			wantCode:      0,
			wantSolutions: 8,
		},
		{
			name:          "limit above the number of solutions",
			args:          []string{"-limit", "100"},
			stdin:         board,
			wantCode:      0,
			wantSolutions: 8,
		},
		{
			name:       "no solution",
			stdin:      conflictingBoard,
			wantCode:   1,
			wantStderr: "sudoku: no solution found\n",
		},
		{
			name:       "invalid board",
			stdin:      "1 2 x",
			wantCode:   2,
			wantStderr: "sudoku: line 1, column 3: expected digit or underscore symbol\n",
		},
		{
			name:       "text after the board",
			stdin:      board + "\nmore",
			wantCode:   2,
			wantStderr: "sudoku: line 10: unexpected content after the board\n",
		},
		{
			name:       "empty input",
			stdin:      "",
			wantCode:   2,
			wantStderr: "sudoku: line 1: unexpected end of input\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			stdin:      board,
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: sudoku",
		},
		{
			name:       "negative limit",
			args:       []string{"-limit", "-1"},
			stdin:      board,
			wantCode:   2,
			wantStderr: `invalid value "-1" for flag -limit`,
		},
		{
			name:       "positional argument",
			args:       []string{"sudoku.txt"},
			stdin:      board,
			wantCode:   2,
			wantStderr: "sudoku: unexpected argument \"sudoku.txt\"\nusage: sudoku",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantCode:   0,
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
			if tt.wantSolutions == 0 && stdout.Len() != 0 {
				t.Errorf("unexpected stdout:\n%s", stdout.String())
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
		name   string
		args   []string
		stdin  string
		failOn int
	}{
		{
			// The board has too many solutions to enumerate, so this ends only
			// if the failure stops the search.
			name:   "header stops an unbounded search",
			args:   []string{"-limit", "0"},
			stdin:  emptyBoard,
			failOn: 1,
		},
		{
			// The header is the first write and each row is one more.
			name:   "middle of the board",
			stdin:  readBoardFile(t),
			failOn: 5,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &failingWriter{failOn: tt.failOn}, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "sudoku: writing solution: write failure\n"
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
