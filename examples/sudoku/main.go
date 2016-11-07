package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dimchansky/dlx/examples/sudoku/parser"
	"github.com/dimchansky/dlx/examples/sudoku/solver"
)

var (
	solutionsLimit = flag.Uint64("limit", 1, "Limit on the number of solutions (specify zero to find all solutions)")
)

func main() {
	flag.Parse()

	s, err := parser.ParseReader(os.Stdin)
	if err != nil {
		fmt.Println(err)
		return
	}

	solutionsFound := uint64(0)
	solver.Solve(s, solver.SolutionAccepterFunc(func(solution string) bool {
		solutionsFound++
		fmt.Printf("Solution %d:\n", solutionsFound)
		printSudoku(solution)
		return *solutionsLimit != 0 && solutionsFound >= *solutionsLimit
	}))

	if solutionsFound == 0 {
		fmt.Println("No solution found")
	}
}

func printSudoku(s string) {
	for r, i := 0, 0; r < 9; r, i = r+1, i+9 {
		fmt.Printf("%c %c %c %c %c %c %c %c %c\n", s[i], s[i+1], s[i+2],
			s[i+3], s[i+4], s[i+5], s[i+6], s[i+7], s[i+8])
	}
}
