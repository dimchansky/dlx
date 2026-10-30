package parser

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

const validBoard = `1 _ 3 _ _ 6 _ 8 _
_ 5 _ _ 8 _ 1 2 _
7 _ 9 1 _ 3 _ 5 6
_ 3 _ _ 6 7 _ 9 _
5 _ 7 8 _ _ _ 3 _
8 _ 1 _ 3 _ 5 _ 7
_ 4 _ _ 7 8 _ 1 _
6 _ 8 _ _ 2 _ 4 _
_ 1 2 _ 4 5 _ 7 8`

const validBoardResult = "1_3__6_8__5__8_12_7_91_3_56_3__67_9_5_78___3_8_1_3_5_7_4__78_1_6_8__2_4__12_45_78"

func TestParseReader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  string
	}{
		{
			name:     "valid board",
			input:    validBoard,
			expected: validBoardResult,
		},
		{
			name:     "windows line endings",
			input:    strings.ReplaceAll(validBoard, "\n", "\r\n"),
			expected: validBoardResult,
		},
		{
			name:     "extra whitespace",
			input:    "  " + strings.ReplaceAll(validBoard, " ", " \t "),
			expected: validBoardResult,
		},
		{
			name:     "blank lines after the board",
			input:    validBoard + "\n\n  \t\n\n",
			expected: validBoardResult,
		},
		{
			name:     "line just under the scanner limit",
			input:    strings.Repeat(" ", bufio.MaxScanTokenSize-1-len("1 _ 3 _ _ 6 _ 8 _")) + validBoard,
			expected: validBoardResult,
		},
		{
			name:     "zero is kept as a blank",
			input:    strings.Replace(validBoard, "_", "0", 1),
			expected: strings.Replace(validBoardResult, "_", "0", 1),
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: "line 1: unexpected end of input",
		},
		{
			name:    "letter in first column",
			input:   "q",
			wantErr: "line 1, column 1: expected digit or underscore symbol",
		},
		{
			name:    "letter in second column",
			input:   "1 q",
			wantErr: "line 1, column 2: expected digit or underscore symbol",
		},
		{
			name:    "dot is not a blank in this format",
			input:   ". 2",
			wantErr: "line 1, column 1: expected digit or underscore symbol",
		},
		{
			name:    "cell with two digits",
			input:   "12",
			wantErr: "line 1, column 1: expected digit or underscore symbol",
		},
		{
			name:    "non-ascii cell",
			input:   "1 é",
			wantErr: "line 1, column 2: expected digit or underscore symbol",
		},
		{
			name:    "short line",
			input:   "1 2 3 4 _",
			wantErr: "line 1: expected 9 cells, got 5",
		},
		{
			name:    "long line",
			input:   "1 2 3 4 5 6 7 8 9 _",
			wantErr: "line 1: too many symbols",
		},
		{
			name: "long second line",
			input: `1 2 3 4 5 6 7 8 9
1 2 3 4 5 6 7 8 9 q`,
			wantErr: "line 2: too many symbols",
		},
		{
			name: "missing lines",
			input: `1 2 _ 4 5 6 7 8 9
1 2 3 4 5 6 7 _ 9`,
			wantErr: "line 3: unexpected end of input",
		},
		{
			name:    "blank line",
			input:   "1 2 3 4 5 6 7 8 9\n\n",
			wantErr: "line 2: expected 9 cells, got 0",
		},
		{
			name:    "content after the board",
			input:   validBoard + "\n1 2 3 4 5 6 7 8 9",
			wantErr: "line 10: unexpected content after the board",
		},
		{
			name:    "content after blank lines",
			input:   validBoard + "\n\n\nnot a board line\n",
			wantErr: "line 12: unexpected content after the board",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseReader(strings.NewReader(tt.input))
			if got != tt.expected {
				t.Errorf("ParseReader() = %q, want %q", got, tt.expected)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr):
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseReader_ReadFailures(t *testing.T) {
	t.Parallel()

	errRead := errors.New("read failure")

	tests := []struct {
		name    string
		input   io.Reader
		wantErr error
		wantMsg string
	}{
		{
			name:    "line beyond the scanner limit",
			input:   strings.NewReader(strings.Repeat(" ", bufio.MaxScanTokenSize)),
			wantErr: bufio.ErrTooLong,
			wantMsg: "line 1: reading input: bufio.Scanner: token too long",
		},
		{
			name:    "reader fails",
			input:   iotest.ErrReader(errRead),
			wantErr: errRead,
			wantMsg: "line 1: reading input: read failure",
		},
		{
			name:    "reader fails after a partial line",
			input:   io.MultiReader(strings.NewReader("1 2 3"), iotest.ErrReader(errRead)),
			wantErr: errRead,
			wantMsg: "line 1: reading input: read failure",
		},
		{
			name: "reader fails after a ninth line without a newline",
			input: io.MultiReader(
				strings.NewReader(strings.Repeat("1 2 3 4 5 6 7 8 9\n", 8)+"1 2 3 4 5 6 7 8 9"),
				iotest.ErrReader(errRead),
			),
			wantErr: errRead,
			wantMsg: "line 9: reading input: read failure",
		},
		{
			name:    "reader fails after the board",
			input:   io.MultiReader(strings.NewReader(validBoard+"\n"), iotest.ErrReader(errRead)),
			wantErr: errRead,
			wantMsg: "line 10: reading input: read failure",
		},
		{
			name:    "reader fails after partial content following the board",
			input:   io.MultiReader(strings.NewReader(validBoard+"\nmore"), iotest.ErrReader(errRead)),
			wantErr: errRead,
			wantMsg: "line 10: reading input: read failure",
		},
		{
			name:    "reader fails after a partial blank line following the board",
			input:   io.MultiReader(strings.NewReader(validBoard+"\n  "), iotest.ErrReader(errRead)),
			wantErr: errRead,
			wantMsg: "line 10: reading input: read failure",
		},
		{
			name:    "line beyond the scanner limit after the board",
			input:   strings.NewReader(validBoard + "\n" + strings.Repeat(" ", bufio.MaxScanTokenSize)),
			wantErr: bufio.ErrTooLong,
			wantMsg: "line 10: reading input: bufio.Scanner: token too long",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseReader(tt.input)
			if got != "" {
				t.Errorf("ParseReader() = %q, want empty", got)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want it to match %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.wantMsg {
				t.Errorf("error = %q, want %q", err, tt.wantMsg)
			}
		})
	}
}
