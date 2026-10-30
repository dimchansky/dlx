// Package parser reads Sudoku boards in the text format of the sudoku example.
package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const boardSize = 9

// ParseReader parses a Sudoku board from r and returns it in 81 character
// string format, row by row. The result is accepted by the solver package.
//
// The board is nine lines of nine whitespace-separated cells. A cell is a digit
// from 1 to 9, or '_' or '0' for a blank. Blanks are returned as given, so the
// result can contain both characters. The input is not checked for conflicting
// digits, and whatever follows the ninth line is not read as part of the board.
//
// ParseReader returns an error naming the line for malformed input, for a read
// failure, and for a line longer than bufio.MaxScanTokenSize, which matches
// bufio.ErrTooLong.
func ParseReader(r io.Reader) (string, error) {
	bs := make([]byte, 0, boardSize*boardSize)

	s := bufio.NewScanner(r)
	for l := 1; l <= boardSize; l++ {
		if !s.Scan() {
			if err := s.Err(); err != nil {
				return "", fmt.Errorf("line %d: reading input: %w", l, err)
			}
			return "", fmt.Errorf("line %d: unexpected end of input", l)
		}

		cells := bytes.Fields(s.Bytes())
		for c := 1; c <= boardSize; c++ {
			if c > len(cells) {
				return "", fmt.Errorf("line %d: unexpected end of input", l)
			}
			t := cells[c-1]
			tc := t[0]
			if len(t) > 1 || ((tc != '_') && ('0' > tc || tc > '9')) {
				return "", fmt.Errorf("line %d, column %d: expected digit or underscore symbol", l, c)
			}
			bs = append(bs, tc)
		}
		if len(cells) > boardSize {
			return "", fmt.Errorf("line %d: too many symbols", l)
		}
	}

	return string(bs), nil
}
