package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const firstTiling = `Tiling 1:
LLXFFTTTWW
LXXXFFTWWN
LUXUFZTWNN
LUUUYZZZNV
PPPYYYYZNV
PPIIIIIVVV
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
			name:        "first tiling by default",
			wantTilings: 1,
			wantStdout:  firstTiling,
		},
		{
			name:        "limit stops the search",
			args:        []string{"-limit", "3"},
			wantTilings: 3,
		},
		{
			name:        "zero limit prints all tilings",
			args:        []string{"-size", "3x20", "-limit", "0"},
			wantTilings: 8,
		},
		{
			name:       "count prints totals",
			args:       []string{"-size", "3x20", "-count"},
			wantStdout: "Tilings: 8\nDistinct: 2\n",
		},
		{
			name:       "count ignores limit",
			args:       []string{"-size", "3x20", "-count", "-limit", "1"},
			wantStdout: "Tilings: 8\nDistinct: 2\n",
		},
		{
			name:       "no tiling",
			args:       []string{"-size", "2x30"},
			wantCode:   1,
			wantStderr: "pentomino: no tiling found\n",
		},
		{
			name:       "count with no tiling",
			args:       []string{"-size", "2x30", "-count"},
			wantCode:   1,
			wantStdout: "Tilings: 0\nDistinct: 0\n",
			wantStderr: "pentomino: no tiling found\n",
		},
		{
			name:       "invalid size",
			args:       []string{"-size", "6x6"},
			wantCode:   2,
			wantStderr: `pentomino: invalid size "6x6": the board must have 60 cells`,
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: pentomino",
		},
		{
			name:       "positional argument",
			args:       []string{"6x10"},
			wantCode:   2,
			wantStderr: "pentomino: unexpected argument \"6x10\"\nusage: pentomino",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: pentomino [-size RxC] [-limit n] [-count]",
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
		args   []string
		failOn int
	}{
		{name: "header stops the search", args: []string{"-limit", "0"}, failOn: 1},
		{name: "middle of a tiling", failOn: 3},
		{name: "totals", args: []string{"-size", "3x20", "-count"}, failOn: 1},
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
			want := "pentomino: writing output: write failure\n"
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
