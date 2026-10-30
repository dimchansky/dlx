package dlx_test

import (
	"fmt"

	"github.com/dimchansky/dlx"
)

func ExampleNewMatrix() {
	m := dlx.NewMatrix(7)
	m.AddRow(0, 3, 6)
	m.AddRow(0, 3)
	m.AddRow(3, 4, 6)
	m.AddRow(2, 4, 5)
	m.AddRow(1, 2, 5, 6)
	m.AddRow(1, 6)

	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		fmt.Println("Solution found:")
		for _, row := range exactCover {
			fmt.Println(row)
		}
		return false
	}))

	// Output:
	// Solution found:
	// [0 3]
	// [2 4 5]
	// [1 6]
}

// Returning true from the accepter stops the search. The matrix stays valid, so
// it can be solved again.
func ExampleMatrix_Solve_earlyStop() {
	m := dlx.NewMatrix(2)
	m.AddRow(0)
	m.AddRow(1)
	m.AddRow(0, 1)

	solutions := 0
	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		solutions++
		fmt.Println("First:", exactCover)
		return true
	}))
	fmt.Println("Solutions seen after the stop:", solutions)

	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		fmt.Println("All:", exactCover)
		return false
	}))

	// Output:
	// First: [[0] [1]]
	// Solutions seen after the stop: 1
	// All: [[0] [1]]
	// All: [[0 1]]
}
