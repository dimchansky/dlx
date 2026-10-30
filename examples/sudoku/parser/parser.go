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
// result can contain both characters. Only blank lines may follow the ninth
// line, so r is read until end of input. The board is not checked for
// conflicting digits.
//
// ParseReader returns an error naming the line for malformed input, for a read
// failure, and for a line of bufio.MaxScanTokenSize bytes or more, which matches
// bufio.ErrTooLong.
func ParseReader(r io.Reader) (string, error) {
	bs := make([]byte, 0, boardSize*boardSize)

	s := bufio.NewScanner(r)
	for l := 1; l <= boardSize; l++ {
		// Err comes first: a reader that fails mid-line still yields the partial
		// line, with Scan returning true and Err already set.
		scanned := s.Scan()
		if err := s.Err(); err != nil {
			return "", fmt.Errorf("line %d: reading input: %w", l, err)
		}
		if !scanned {
			return "", fmt.Errorf("line %d: unexpected end of input", l)
		}

		cells := bytes.Fields(s.Bytes())
		for c := 0; c < boardSize && c < len(cells); c++ {
			t := cells[c]
			tc := t[0]
			if len(t) > 1 || ((tc != '_') && ('0' > tc || tc > '9')) {
				return "", fmt.Errorf("line %d, column %d: expected digit or underscore symbol", l, c+1)
			}
			bs = append(bs, tc)
		}
		if len(cells) < boardSize {
			return "", fmt.Errorf("line %d: expected %d cells, got %d", l, boardSize, len(cells))
		}
		if len(cells) > boardSize {
			return "", fmt.Errorf("line %d: too many symbols", l)
		}
	}

	for l := boardSize + 1; ; l++ {
		scanned := s.Scan()
		if err := s.Err(); err != nil {
			return "", fmt.Errorf("line %d: reading input: %w", l, err)
		}
		if !scanned {
			break
		}
		if len(bytes.TrimSpace(s.Bytes())) != 0 {
			return "", fmt.Errorf("line %d: unexpected content after the board", l)
		}
	}

	return string(bs), nil
}
