package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantLines  int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "first series by default",
			wantLines:  1,
			wantStdout: "C C# G# B E A# G A F D# D F#\n",
		},
		{
			name:       "every series for six pitches",
			args:       []string{"-n", "6", "-limit", "0"},
			wantLines:  4,
			wantStdout: "0 1 5 2 4 3\n0 2 1 4 5 3\n0 4 5 2 1 3\n0 5 1 4 2 3\n",
		},
		{
			name:       "other sizes print numbers",
			args:       []string{"-n", "4", "-limit", "0"},
			wantLines:  2,
			wantStdout: "0 1 3 2\n0 3 1 2\n",
		},
		{
			name:       "first pitch moves the start",
			args:       []string{"-n", "4", "-first", "1", "-limit", "0"},
			wantLines:  2,
			wantStdout: "1 0 2 3\n1 2 0 3\n",
		},
		{
			name:      "every first pitch",
			args:      []string{"-n", "4", "-first", "-1", "-limit", "0"},
			wantLines: 8,
		},
		{
			name:      "limit stops the search",
			args:      []string{"-limit", "3"},
			wantLines: 3,
		},
		{
			name:      "limit stops the search across first pitches",
			args:      []string{"-n", "4", "-first", "-1", "-limit", "3"},
			wantLines: 3,
		},
		{
			name:      "zero limit prints all series",
			args:      []string{"-n", "8", "-limit", "0"},
			wantLines: 24,
		},
		{
			name:       "count prints the total",
			args:       []string{"-n", "8", "-count"},
			wantLines:  1,
			wantStdout: "Series: 24\n",
		},
		{
			name:       "count over every first pitch",
			args:       []string{"-n", "8", "-first", "-1", "-count"},
			wantLines:  1,
			wantStdout: "Series: 192\n",
		},
		{
			name:       "count ignores limit",
			args:       []string{"-n", "8", "-count", "-limit", "1"},
			wantLines:  1,
			wantStdout: "Series: 24\n",
		},
		{
			name:       "smallest size",
			args:       []string{"-n", "2"},
			wantLines:  1,
			wantStdout: "0 1\n",
		},
		{
			name:       "no series",
			args:       []string{"-n", "5"},
			wantCode:   1,
			wantStderr: "twelvetone: no series found\n",
		},
		{
			name:       "count with no series",
			args:       []string{"-n", "5", "-count"},
			wantCode:   1,
			wantLines:  1,
			wantStdout: "Series: 0\n",
			wantStderr: "twelvetone: no series found\n",
		},
		{
			name:       "size too small",
			args:       []string{"-n", "1"},
			wantCode:   2,
			wantStderr: "twelvetone: invalid size 1: want 2 to 32\n",
		},
		{
			name:       "size too large",
			args:       []string{"-n", "33"},
			wantCode:   2,
			wantStderr: "twelvetone: invalid size 33: want 2 to 32\n",
		},
		{
			name:       "first pitch too small",
			args:       []string{"-first", "-2"},
			wantCode:   2,
			wantStderr: "twelvetone: invalid first pitch -2: want -1 to 11\n",
		},
		{
			name:       "first pitch too large",
			args:       []string{"-n", "4", "-first", "4"},
			wantCode:   2,
			wantStderr: "twelvetone: invalid first pitch 4: want -1 to 3\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: twelvetone",
		},
		{
			name:       "positional argument",
			args:       []string{"12"},
			wantCode:   2,
			wantStderr: "twelvetone: unexpected argument \"12\"\nusage: twelvetone",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: twelvetone [-n size] [-first pitch] [-limit n] [-count]",
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
			if got := strings.Count(stdout.String(), "\n"); got != tt.wantLines {
				t.Errorf("printed %d lines, want %d", got, tt.wantLines)
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

// TestRun_ShiftedSeries runs the program from Berg's starting pitch and checks
// that it prints exactly the series from pitch 0, shifted up, and Berg's among
// them. This is the only dlx run from another pitch at this size.
func TestRun_ShiftedSeries(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow: enumerates every series")
	}

	const n, first = 12, 5
	want := make(map[string]bool)
	for _, series := range seriesFromZero(n) {
		shifted := make([]int, n)
		for i, pitch := range series {
			shifted[i] = (pitch + first) % n
		}
		want[formatSeries(shifted)] = true
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-first", "5", "-limit", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit status %d, stderr: %s", code, stderr.String())
	}

	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 3856 {
		t.Errorf("printed %d lines, want 3856", len(lines))
	}
	got := make(map[string]bool)
	for _, line := range lines {
		got[line] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("printed series are not the shifted series from pitch 0")
	}
	if !got["F E C A G D G# C# D# F# A# B"] {
		t.Errorf("Berg's series is missing")
	}
}

func TestRun_WriteFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		failOn int
	}{
		{name: "first line stops the search", args: []string{"-n", "8", "-limit", "0"}, failOn: 1},
		{name: "later line stops the search", args: []string{"-n", "8", "-limit", "0"}, failOn: 5},
		{name: "total", args: []string{"-n", "8", "-count"}, failOn: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			writer := &failingWriter{failOn: tt.failOn}
			code := run(tt.args, writer, &stderr)

			if code != 2 {
				t.Errorf("exit status %d, want 2", code)
			}
			want := "twelvetone: writing output: write failure\n"
			if stderr.String() != want {
				t.Errorf("stderr = %q, want %q", stderr.String(), want)
			}
			if writer.writes != tt.failOn {
				t.Errorf("made %d writes, want the search to stop after %d", writer.writes, tt.failOn)
			}
		})
	}
}

func TestFormatSeries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		series []int
		want   string
	}{
		{name: "twelve pitches are notes", series: bergRow, want: "F E C A G D G# C# D# F# A# B"},
		{name: "other sizes are numbers", series: []int{0, 3, 1, 2}, want: "0 3 1 2"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := formatSeries(tt.series); got != tt.want {
				t.Errorf("formatSeries(%v) = %q, want %q", tt.series, got, tt.want)
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
