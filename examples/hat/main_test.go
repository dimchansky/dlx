package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "default disk uses both shapes",
			wantStdout: "Disk of radius 6: 72 kites, 240 hat placements\nCover: 18 hats, 3 of them reflected\n",
		},
		{
			name:       "largest disk with rotations only",
			args:       []string{"-r", "5", "-rotations"},
			wantStdout: "Disk of radius 5: 48 kites, 78 hat placements\nCover: 11 hats, 0 of them reflected\n",
		},
		{
			name:       "rotations alone fail",
			args:       []string{"-rotations"},
			wantCode:   1,
			wantStdout: "Disk of radius 6: 72 kites, 120 hat placements\n",
			wantStderr: "hat: no cover found\n",
		},
		{
			name:       "count",
			args:       []string{"-r", "5", "-rotations", "-count"},
			wantStdout: "Covers: 14\n",
		},
		{
			name:       "count with no cover",
			args:       []string{"-rotations", "-count"},
			wantCode:   1,
			wantStdout: "Covers: 0\n",
			wantStderr: "hat: no cover found\n",
		},
		{
			name:       "picture with no cover",
			args:       []string{"-rotations", "-svg"},
			wantCode:   1,
			wantStderr: "hat: no cover found\n",
		},
		{
			name:       "radius too small",
			args:       []string{"-r", "1"},
			wantCode:   2,
			wantStderr: "hat: invalid radius 1: want 2 to 64\n",
		},
		{
			name:       "radius too large",
			args:       []string{"-r", "65"},
			wantCode:   2,
			wantStderr: "hat: invalid radius 65: want 2 to 64\n",
		},
		{
			name:       "count and picture together",
			args:       []string{"-count", "-svg"},
			wantCode:   2,
			wantStderr: "hat: -count and -svg cannot be used together\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: hat",
		},
		{
			name:       "positional argument",
			args:       []string{"6"},
			wantCode:   2,
			wantStderr: "hat: unexpected argument \"6\"\nusage: hat",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: hat [-r radius] [-rotations] [-count] [-svg]",
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

// TestRun_Pictures keeps the pictures in the README equal to the program's
// output.
func TestRun_Pictures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file string
		args []string
	}{
		{file: "small.svg", args: []string{"-r", "2", "-rotations", "-svg"}},
		{file: "rotations.svg", args: []string{"-r", "5", "-rotations", "-svg"}},
		{file: "mirrors.svg", args: []string{"-svg"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.file, func(t *testing.T) {
			t.Parallel()

			want, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			if code := run(tt.args, &stdout, &stderr); code != 0 {
				t.Fatalf("exit status %d, stderr: %s", code, stderr.String())
			}
			if !bytes.Equal(stdout.Bytes(), want) {
				t.Errorf("%s differs from the output of hat %v", tt.file, tt.args)
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
		{name: "summary", args: []string{"-r", "2"}, failOn: 1},
		{name: "cover line", args: []string{"-r", "2"}, failOn: 2},
		{name: "total", args: []string{"-r", "2", "-count"}, failOn: 1},
		{name: "picture", args: []string{"-r", "2", "-svg"}, failOn: 1},
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
			want := "hat: writing output: write failure\n"
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
