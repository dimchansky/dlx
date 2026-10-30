package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const firstCode = `Code 1:
0000000
0000111
0011001
0011110
0101010
0101101
0110011
0110100
1001011
1001100
1010010
1010101
1100001
1100110
1111000
1111111
`

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string // a substring, because usage text may follow
	}{
		{name: "first code by default", wantStdout: firstCode},
		{
			name:       "shortest words",
			args:       []string{"-n", "1"},
			wantStdout: "Code 1:\n0\n",
		},
		{
			name:       "limit two",
			args:       []string{"-n", "3", "-limit", "2"},
			wantStdout: "Code 1:\n000\n111\nCode 2:\n001\n110\n",
		},
		{
			name:       "ternary first code",
			args:       []string{"-base", "3", "-n", "4"},
			wantStdout: "Code 1:\n0000\n0111\n0222\n1012\n1120\n1201\n2021\n2102\n2210\n",
		},
		{name: "count", args: []string{"-count"}, wantStdout: "Codes: 240\n"},
		{name: "count ignores limit", args: []string{"-count", "-limit", "1"}, wantStdout: "Codes: 240\n"},
		{name: "ternary count", args: []string{"-base", "3", "-n", "4", "-count"}, wantStdout: "Codes: 72\n"},
		{
			name:       "no code of that length",
			args:       []string{"-n", "8"},
			wantCode:   1,
			wantStderr: "hamming: no perfect code found\n",
		},
		{
			name:       "count with no code",
			args:       []string{"-n", "14", "-count"},
			wantCode:   1,
			wantStdout: "Codes: 0\n",
			wantStderr: "hamming: no perfect code found\n",
		},
		{
			name:       "hat score",
			args:       []string{"-hat"},
			wantStdout: firstCode + "Hat patterns:  128\nTeam wins:     112\nTeam loses:    16\nRight guesses: 112\nWrong guesses: 112\n",
		},
		{
			name: "round won",
			args: []string{"-n", "3", "-play", "001"},
			wantStdout: "Code 1:\n000\n111\nHats: 001\n" +
				"Prisoner 1 sees ?01 and passes\n" +
				"Prisoner 2 sees 0?1 and passes\n" +
				"Prisoner 3 sees 00? and says 1: right\n" +
				"Team wins\n",
		},
		{
			name: "round lost",
			args: []string{"-n", "3", "-play", "111"},
			wantStdout: "Code 1:\n000\n111\nHats: 111\n" +
				"Prisoner 1 sees ?11 and says 0: wrong\n" +
				"Prisoner 2 sees 1?1 and says 0: wrong\n" +
				"Prisoner 3 sees 11? and says 0: wrong\n" +
				"Team loses\n",
		},
		{name: "hat with no code", args: []string{"-n", "4", "-hat"}, wantCode: 1, wantStderr: "hamming: no perfect code found\n"},
		{name: "bad base", args: []string{"-base", "4"}, wantCode: 2, wantStderr: "invalid base 4: want 2 or 3"},
		{name: "length too small", args: []string{"-n", "0"}, wantCode: 2, wantStderr: "invalid length 0: want 1 to 14 for base 2"},
		{name: "length too large", args: []string{"-n", "15"}, wantCode: 2, wantStderr: "invalid length 15: want 1 to 14 for base 2"},
		{name: "ternary length too large", args: []string{"-base", "3", "-n", "9"}, wantCode: 2, wantStderr: "invalid length 9: want 1 to 8 for base 3"},
		{name: "stray argument", args: []string{"extra"}, wantCode: 2, wantStderr: `unexpected argument "extra"`},
		{name: "unknown flag", args: []string{"-x"}, wantCode: 2, wantStderr: "flag provided but not defined"},
		{name: "hat with count", args: []string{"-hat", "-count"}, wantCode: 2, wantStderr: "use only one of"},
		{name: "hat with play", args: []string{"-hat", "-play", "0000000"}, wantCode: 2, wantStderr: "use only one of"},
		{name: "play with count", args: []string{"-count", "-play", "0000000"}, wantCode: 2, wantStderr: "use only one of"},
		{name: "hat needs base 2", args: []string{"-base", "3", "-n", "4", "-hat"}, wantCode: 2, wantStderr: "needs base 2"},
		{name: "hats too short", args: []string{"-play", "01"}, wantCode: 2, wantStderr: `invalid hats "01": want 7 digits`},
		{name: "hats not binary", args: []string{"-n", "3", "-play", "012"}, wantCode: 2, wantStderr: `invalid hats "012"`},
		{name: "empty hats", args: []string{"-play="}, wantCode: 2, wantStderr: `invalid hats "": want 7 digits`},
		{name: "empty hats with count", args: []string{"-play=", "-count"}, wantCode: 2, wantStderr: "use only one of"},
		{name: "empty hats with hat", args: []string{"-play=", "-hat"}, wantCode: 2, wantStderr: "use only one of"},
		{name: "empty hats need base 2", args: []string{"-play=", "-base", "3", "-n", "4"}, wantCode: 2, wantStderr: "needs base 2"},
		{name: "help", args: []string{"-help"}, wantStderr: "usage: hamming"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code %d, want %d; stderr: %s", code, tt.wantCode, stderr.String())
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr == "" && stderr.Len() != 0 {
				t.Errorf("unexpected stderr: %s", stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

// TestPrintRound_WinRule plays plans that are not perfect codes, the only ones
// that can produce these two losses.
func TestPrintRound_WinRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code []int
		hats string
		want string
	}{
		{
			name: "everybody passes",
			hats: "000",
			want: "Hats: 000\n" +
				"Prisoner 1 sees ?00 and passes\n" +
				"Prisoner 2 sees 0?0 and passes\n" +
				"Prisoner 3 sees 00? and passes\n" +
				"Team loses\n",
		},
		{
			name: "right and wrong guesses mixed",
			code: []int{0, 1},
			hats: "001",
			want: "Hats: 001\n" +
				"Prisoner 1 sees ?01 and says 1: wrong\n" +
				"Prisoner 2 sees 0?1 and says 1: wrong\n" +
				"Prisoner 3 sees 00? and says 1: right\n" +
				"Team loses\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			if err := printRound(&out, tt.code, 3, tt.hats); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", out.String(), tt.want)
			}
		})
	}
}

// failingWriter fails after accepting a set number of writes.
type failingWriter struct{ writesLeft int }

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.writesLeft == 0 {
		return 0, errors.New("disk full")
	}
	w.writesLeft--
	return len(p), nil
}

func TestRun_WriteFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		goodWrites int
	}{
		{name: "code header", goodWrites: 0},
		{name: "code word", goodWrites: 1},
		{name: "count", args: []string{"-count"}},
		{name: "hat code", args: []string{"-hat"}},
		{name: "hat score", args: []string{"-hat"}, goodWrites: 17},
		{name: "round hats", args: []string{"-play", "0000000"}, goodWrites: 17},
		{name: "round prisoner", args: []string{"-play", "0000000"}, goodWrites: 18},
		{name: "round verdict", args: []string{"-play", "0000000"}, goodWrites: 25},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := run(tt.args, &failingWriter{writesLeft: tt.goodWrites}, &stderr)
			if code != 2 {
				t.Errorf("exit code %d, want 2", code)
			}
			if !strings.Contains(stderr.String(), "writing output: disk full") {
				t.Errorf("unexpected stderr: %s", stderr.String())
			}
		})
	}
}
