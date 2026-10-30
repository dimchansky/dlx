package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const firstExtent = `Extent 1:
1234
2134
2143
1243
1423
4123
4213
2413
2431
4231
4321
4312
4132
1432
1342
3142
3412
3421
3241
2341
2314
3214
3124
1324
1234
`

func TestRun(t *testing.T) {
	t.Parallel()

	const threeBells = "Extent 1:\n123\n213\n231\n321\n312\n132\n123\n"

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string // a substring, because usage text may follow
		slow       bool   // skipped under -short: goes through every cover of abc
	}{
		{name: "first extent by default", wantStdout: firstExtent},
		{name: "three bells", args: []string{"-bells", "3"}, wantStdout: threeBells},
		{
			name:       "zero limit prints all extents",
			args:       []string{"-bells", "3", "-limit", "0"},
			wantStdout: threeBells + "Extent 2:\n123\n132\n312\n321\n231\n213\n123\n",
		},
		{name: "count", args: []string{"-count"}, wantStdout: "Cycle covers: 28561\nExtents: 88\n", slow: true},
		{
			name:       "count ignores limit",
			args:       []string{"-bells", "3", "-count", "-limit", "1"},
			wantStdout: "Cycle covers: 4\nExtents: 2\n",
		},
		{
			name:       "order and repeats of letters do not matter",
			args:       []string{"-bells", "3", "-count", "-changes", "baa"},
			wantStdout: "Cycle covers: 4\nExtents: 2\n",
		},
		{
			name:       "double swap",
			args:       []string{"-count", "-changes", "bd"},
			wantStdout: "Cycle covers: 64\nExtents: 0\n",
			wantCode:   1,
			wantStderr: "bells: no extent found\n",
		},
		{
			name:       "no extent",
			args:       []string{"-changes", "ab"},
			wantCode:   1,
			wantStderr: "bells: no extent found\n",
		},
		{
			name:       "count with no extent",
			args:       []string{"-changes", "ab", "-count"},
			wantCode:   1,
			wantStdout: "Cycle covers: 256\nExtents: 0\n",
			wantStderr: "bells: no extent found\n",
		},
		{
			name:       "too few bells",
			args:       []string{"-bells", "2"},
			wantCode:   2,
			wantStderr: "bells: invalid number of bells 2: want 3 or 4\n",
		},
		{
			name:       "too many bells",
			args:       []string{"-bells", "6"},
			wantCode:   2,
			wantStderr: "bells: invalid number of bells 6: want 3 or 4\n",
		},
		{
			name:       "unknown change",
			args:       []string{"-changes", "abx"},
			wantCode:   2,
			wantStderr: "bells: invalid changes \"abx\": want letters from abcd\n",
		},
		{
			name:       "explicitly empty changes",
			args:       []string{"-changes="},
			wantCode:   2,
			wantStderr: "bells: invalid changes \"\": want letters from abcd\n",
		},
		{
			name:       "change that needs a fourth bell",
			args:       []string{"-bells", "3", "-changes", "ad"},
			wantCode:   2,
			wantStderr: "bells: change d needs 4 bells\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: bells",
		},
		{
			name:       "positional argument",
			args:       []string{"4"},
			wantCode:   2,
			wantStderr: "bells: unexpected argument \"4\"\nusage: bells",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: bells [-bells 3|4] [-changes letters] [-limit n] [-count]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.slow && testing.Short() {
				t.Skip("slow: goes through every cycle cover")
			}

			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit status %d, want %d", code, tt.wantCode)
			}
			if stdout.String() != tt.wantStdout {
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
		{name: "rows", failOn: 2},
		{name: "totals", args: []string{"-count", "-changes", "ab"}, failOn: 1},
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
			want := "bells: writing output: write failure\n"
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
