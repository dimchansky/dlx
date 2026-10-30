package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestParse(t *testing.T) {
	t.Parallel()

	input := `# a comment

flights F1 F2 F3
  # an indented comment
A 10 F3 F1
B 0 F2
`
	want := problem{
		flights: []string{"F1", "F2", "F3"},
		pairings: []pairing{
			{name: "A", cost: 10, flights: []int{0, 2}},
			{name: "B", cost: 0, flights: []int{1}},
		},
	}

	got, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "empty input", input: "", wantErr: "no flights line"},
		{name: "only comments", input: "# nothing\n\n", wantErr: "no flights line"},
		{name: "pairing before flights", input: "A 1 F1\n", wantErr: `line 1: expected a line starting with "flights"`},
		{name: "no flight names", input: "flights\n", wantErr: "line 1: expected at least one flight"},
		{name: "repeated flight", input: "flights F1 F1\n", wantErr: `line 1: flight "F1" listed twice`},
		{name: "short pairing", input: "flights F1\nA 1\n", wantErr: "line 2: expected a name, a cost and at least one flight"},
		{name: "repeated name", input: "flights F1\nA 1 F1\nA 2 F1\n", wantErr: `line 3: pairing "A" listed twice`},
		{name: "cost not a number", input: "flights F1\nA x F1\n", wantErr: `line 2: invalid cost "x"`},
		{name: "negative cost", input: "flights F1\nA -1 F1\n", wantErr: `line 2: invalid cost "-1"`},
		{name: "cost too large", input: "flights F1\nA 1000001 F1\n", wantErr: `line 2: invalid cost "1000001"`},
		{name: "unknown flight", input: "flights F1\nA 1 F2\n", wantErr: `line 2: unknown flight "F2"`},
		{name: "flight twice in pairing", input: "flights F1\nA 1 F1 F1\n", wantErr: `line 2: flight "F1" listed twice in pairing "A"`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parse(strings.NewReader(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestParse_ReadFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("read failure")
	_, err := parse(iotest.ErrReader(failure))
	if !errors.Is(err, failure) {
		t.Errorf("error = %v, want it to wrap %v", err, failure)
	}
}
