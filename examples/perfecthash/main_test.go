package main

import (
	"bytes"
	"errors"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// keywordOutput and fiveKeyOutput are the outputs shown in the README.
const keywordOutput = `Keys: 25, first letters: 13
Matrix: 38 columns, 325 rows

Letter  Shift
b       0
c       0
d       2
e       0
f       0
g       2
i       1
m       3
p       8
r       3
s       3
t       4
v       22

Slot  Key
0     var
1     const
2     func
3     chan
4     import
5     default
6     else
7     case
8     fallthrough
9     goto
10    struct
11    package
12    break
13    switch
14    go
15    for
16    map
17    type
18    range
19    defer
20    interface
21    continue
22    if
23    select
24    return
`

const fiveKeyOutput = `Keys: 5, first letters: 4
Matrix: 9 columns, 20 rows

Letter  Shift
f       0
i       0
m       0
v       1

Slot  Key
0     for
1     if
2     func
3     map
4     var
`

// oneKeyOutput has a single row, so the count mustn't say "1 rows".
const oneKeyOutput = `Keys: 1, first letters: 1
Matrix: 2 columns, 1 row

Letter  Shift
a       0

Slot  Key
0     a
`

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "go keywords by default", wantStdout: keywordOutput},
		{name: "own keys", args: []string{"-keys", "for,func,if,map,var"}, wantStdout: fiveKeyOutput},
		{name: "one key", args: []string{"-keys", "a"}, wantStdout: oneKeyOutput},
		{
			name:       "same letter clash",
			args:       []string{"-keys", "ab,ad"},
			wantCode:   1,
			wantStderr: "perfecthash: no hash table found\n",
		},
		{
			name:       "no shifts fit together",
			args:       []string{"-keys", "aa,ab,ba,baa"},
			wantCode:   1,
			wantStderr: "perfecthash: no hash table found\n",
		},
		{
			name:       "uppercase key",
			args:       []string{"-keys", "go,Goto"},
			wantCode:   2,
			wantStderr: "perfecthash: invalid key \"Goto\": want lowercase letters a to z\n",
		},
		{
			name:       "empty key",
			args:       []string{"-keys", "go,,if"},
			wantCode:   2,
			wantStderr: "perfecthash: empty key\n",
		},
		{
			name:       "duplicate key",
			args:       []string{"-keys", "go,if,go"},
			wantCode:   2,
			wantStderr: "perfecthash: duplicate key \"go\"\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"-unknown"},
			wantCode:   2,
			wantStderr: "flag provided but not defined: -unknown\nusage: perfecthash",
		},
		{
			name:       "positional argument",
			args:       []string{"go"},
			wantCode:   2,
			wantStderr: "perfecthash: unexpected argument \"go\"\nusage: perfecthash",
		},
		{
			name:       "help",
			args:       []string{"-help"},
			wantStderr: "usage: perfecthash [-keys word,word,...]",
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

func TestRun_WriteFailure(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	code := run(nil, failingWriter{}, &stderr)

	if code != 2 {
		t.Errorf("exit status %d, want 2", code)
	}
	want := "perfecthash: writing output: write failure\n"
	if stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failure")
}

func TestGoKeywordsMatchGoToken(t *testing.T) {
	t.Parallel()

	var want []string
	for tok := token.Token(0); tok < 1024; tok++ {
		if tok.IsKeyword() {
			want = append(want, tok.String())
		}
	}
	sort.Strings(want)

	got := append([]string(nil), goKeywords...)
	sort.Strings(got)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("goKeywords = %v, go/token has %v", got, want)
	}
	if len(want) != 25 {
		t.Errorf("go/token has %d keywords, the README says 25", len(want))
	}
}

func TestValidateKeys(t *testing.T) {
	t.Parallel()

	tooMany := make([]string, maxKeys+1)
	for i := range tooMany {
		tooMany[i] = string([]byte{'a' + byte(i/26), 'a' + byte(i%26)})
	}

	tests := []struct {
		name    string
		keys    []string
		wantErr string
	}{
		{name: "one letter", keys: []string{"a"}},
		{name: "longest key", keys: []string{strings.Repeat("z", maxKeyLength)}},
		{name: "most keys", keys: tooMany[:maxKeys]},
		{name: "too many keys", keys: tooMany, wantErr: "too many keys: 257, want at most 256"},
		{
			name:    "key too long",
			keys:    []string{strings.Repeat("z", maxKeyLength+1)},
			wantErr: "too long",
		},
		{name: "digit", keys: []string{"go2"}, wantErr: "invalid key \"go2\""},
		{name: "space", keys: []string{"go "}, wantErr: "invalid key \"go \""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateKeys(tt.keys)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateKeys: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validateKeys error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
