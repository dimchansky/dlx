package solver

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/dimchansky/dlx/examples/sudoku/parser"
)

const (
	multiSolutionBoard = ".8...9743.5...8.1..1.......8....5......8.4......3....6.......7..3.5...8.9724...5."
	// conflictingBoard repeats the digit 5 in its first row.
	conflictingBoard = "55..............................................................................."
	solvedBoard      = "857349621432861597619752843271583964945126378386497215768915432194238756523674189"
)

var multiSolutionBoardSolutions = []string{
	"286159743457638219319742568821965437693874125745321896568213974134597682972486351",
	"286159743357648219419723568821965437693874125745312896568231974134597682972486351",
	"286159743357648219419732568821965437693874125745321896568213974134597682972486351",
	"286159743457638912319742568891265437623874195745391826568913274134527689972486351",
	"286159743357648912419732568891265437623874195745391826568913274134527689972486351",
	"286159743354768912719243568823615497697824135145397826568931274431572689972486351",
	"286159743354768912719243568893615427627894135145327896568931274431572689972486351",
	"286159743354768912719243568893615427627894135145372896568931274431527689972486351",
}

var solverTable = []struct {
	name      string
	input     string
	solutions []string
}{
	{
		name:  "seventeen givens",
		input: ".......2143.......6........2.15..........637...........68...4.....23........7....",
		solutions: []string{
			solvedBoard,
		},
	},
	{
		name:  "sparse givens",
		input: "85...24..72......9..4.........1.7..23.5...9...4...........8..7..17..........36.4.",
		solutions: []string{
			"859612437723854169164379528986147352375268914241593786432981675617425893598736241",
		},
	},
	{
		name:  "dense givens",
		input: "....74316...6.384......85..7258...34....3..5......2798..894.....4..859..971326485",
		solutions: []string{
			"589274316217653849463198527725819634896437152134562798658941273342785961971326485",
		},
	},
	{
		name:      "several solutions",
		input:     multiSolutionBoard,
		solutions: multiSolutionBoardSolutions,
	},
	{
		name:  "no solutions",
		input: ".....5.8....6.1.43..........1.5........1.6...3.......553.....61........4.........",
	},
	{
		name:  "conflict in a row",
		input: conflictingBoard,
	},
	{
		name:  "conflict in a column",
		input: "5........5.......................................................................",
	},
	{
		name:  "conflict in a region",
		input: "5.........5......................................................................",
	},
	{
		name:      "already solved",
		input:     solvedBoard,
		solutions: []string{solvedBoard},
	},
	{
		name:  "solved but for a wrong digit",
		input: "1" + solvedBoard[1:],
	},
}

func TestSolve(t *testing.T) {
	t.Parallel()

	for _, tt := range solverTable {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toString(allSolutions(tt.input))
			if want := toString(tt.solutions); got != want {
				t.Errorf("failed to solve %s\ngot:\n%s\nwant:\n%s", tt.input, got, want)
			}
		})
	}
}

func TestSolve_BoardFile(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join("..", "sudoku.txt"))
	if err != nil {
		t.Fatal(err)
	}

	board, err := parser.ParseReader(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	got := toString(allSolutions(board))
	want := toString(multiSolutionBoardSolutions)
	if got != want {
		t.Errorf("wrong solutions\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSolve_BlankCharactersAreEquivalent(t *testing.T) {
	t.Parallel()

	want := toString(multiSolutionBoardSolutions)
	for _, blank := range []string{"_", "0"} {
		blank := blank
		t.Run(blank, func(t *testing.T) {
			t.Parallel()

			board := strings.ReplaceAll(multiSolutionBoard, ".", blank)
			if got := toString(allSolutions(board)); got != want {
				t.Errorf("wrong solutions\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestSolve_StopsWhenAccepterReturnsTrue(t *testing.T) {
	t.Parallel()

	calls := 0
	Solve(multiSolutionBoard, SolutionAccepterFunc(func(string) bool {
		calls++
		return calls == 3
	}))
	if calls != 3 {
		t.Errorf("accepter called %d times, want 3", calls)
	}
}

func TestSolve_InvalidArgumentsPanic(t *testing.T) {
	t.Parallel()

	accepter := SolutionAccepterFunc(func(string) bool { return false })

	tests := []struct {
		name     string
		board    string
		accepter SolutionAccepter
		want     string
	}{
		{
			name:     "empty board",
			board:    "",
			accepter: accepter,
			want:     "solver: board has 0 bytes, want 81",
		},
		{
			name:     "short board",
			board:    multiSolutionBoard[:80],
			accepter: accepter,
			want:     "solver: board has 80 bytes, want 81",
		},
		{
			name:     "long board",
			board:    multiSolutionBoard + ".",
			accepter: accepter,
			want:     "solver: board has 82 bytes, want 81",
		},
		{
			name:     "letter in a cell",
			board:    "x" + multiSolutionBoard[1:],
			accepter: accepter,
			want:     "solver: invalid cell 'x' at position 0",
		},
		{
			name:     "space in a cell",
			board:    multiSolutionBoard[:80] + " ",
			accepter: accepter,
			want:     "solver: invalid cell ' ' at position 80",
		},
		{
			name:     "nil accepter",
			board:    multiSolutionBoard,
			accepter: nil,
			want:     "solver: nil SolutionAccepter",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if got := recover(); got != tt.want {
					t.Errorf("panic value %#v, want %q", got, tt.want)
				}
			}()
			Solve(tt.board, tt.accepter)
		})
	}
}

func TestSolveFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single solution",
			input: solverTable[0].input,
			want:  solverTable[0].solutions,
		},
		{
			name:  "several solutions",
			input: multiSolutionBoard,
			want:  multiSolutionBoardSolutions,
		},
		{
			name:  "no solutions",
			input: conflictingBoard,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := SolveFirst(tt.input)
			if ok != (len(tt.want) > 0) {
				t.Fatalf("SolveFirst() ok = %v with result %q", ok, got)
			}
			if !ok {
				if got != "" {
					t.Errorf("SolveFirst() = %q without a solution, want empty", got)
				}
				return
			}
			if !contains(tt.want, got) {
				t.Errorf("SolveFirst() = %q, want one of %q", got, tt.want)
			}
		})
	}
}

func BenchmarkSolveTestSudoku(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for _, tc := range solverTable {
			Solve(tc.input, SolutionAccepterFunc(func(solution string) bool {
				return false
			}))
		}
	}
}

func allSolutions(board string) []string {
	solutions := []string{}
	Solve(board, SolutionAccepterFunc(func(solution string) bool {
		solutions = append(solutions, solution)
		return false
	}))
	return solutions
}

func toString(solutions []string) string {
	sorted := append([]string{}, solutions...)
	sort.Strings(sorted)
	return strings.Join(sorted, ",\n")
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
