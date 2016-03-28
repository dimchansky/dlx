/*
Package dlx implements the Knuth's algorithm DLX (algorithm X implemented in terms of dancing links).

A fantastic explanation of dancing links algorithm can be found here: http://arxiv.org/pdf/cs/0011047v1.pdf

Example

Consider the exact cover problem specified by the matrix:

	1	0	0	1	0	0	1
    1	0	0	1	0	0	0
    0	0	0	1	1	0	1
    0	0	1	0	1	1	0
    0	1	1	0	0	1	1
    0	1	0	0	0	0	1

The following code finds all solutions to the exact cover problem:

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

The code output:

    Solution found:
    [0 3]
    [2 4 5]
    [1 6]

*/
package dlx
