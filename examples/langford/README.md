# Langford pairings

A command-line solver for Langford pairings, written as an example of using package [dlx](https://github.com/dimchansky/dlx). Take two copies of each number from 1 to n and arrange them in a row, so that the two copies of a number k have exactly k numbers between them. For n=3, one answer is:

    3 1 2 1 3 2

The two 1s have one number between them, the two 2s have two, and the two 3s have three.

## How it maps to exact cover

The encoding is about as small as exact cover gets, which leaves room to show a different idea: removing symmetry from the matrix.

| Matrix part | Meaning                                                  | Count for n=7             |
|-------------|----------------------------------------------------------|---------------------------|
| Column      | A number, which must be placed exactly once              | 7                         |
| Column      | A position, which must be filled exactly once            | 14                        |
| Row         | A number placed at one pair of positions                 | 63, or 57 without mirrors |

Each row has three 1s: the number, its first position, and its second position, which is number+1 places later.

## Removing mirror images

The reversal of an arrangement is also an arrangement, and it is a different exact cover. So every solution has a mirror twin, and the plain matrix finds both.

You can filter the twins afterwards, as the [pentomino example](../pentomino) does, but the search has then already done the work. Removing rows from the matrix is better, because the search never enters those branches.

Which rows? Take the number 1, with positions counted from 0. If it sits at positions p and p+2, then in the reversal it sits at 2n-3-p and 2n-1-p. The first positions p and 2n-3-p add up to an odd number, so they are never equal, and exactly one of them is below n-1. Keeping only the rows that place 1 there keeps exactly one arrangement of each twin pair. A number with this property has to be odd: an even number can sit symmetrically around the centre, so its row would be its own mirror.

[solver.go](solver.go) does this unless you pass `-mirrors`. The tests confirm that the reduced matrix finds half as many arrangements, one of each pair.

| n             | 1 | 2 | 3 | 4 | 5 | 6 | 7  | 8   | 9 | 10 | 11    | 12     |
|---------------|---|---|---|---|---|---|----|-----|---|----|-------|--------|
| Up to mirrors | 0 | 0 | 1 | 1 | 0 | 0 | 26 | 150 | 0 | 0  | 17792 | 108144 |

With `-mirrors`, each total doubles. Arrangements exist only when n leaves a remainder of 0 or 3 when divided by 4.

The tests check these totals and compare every arrangement with an independent backtracking search.

## Installation

    go install github.com/dimchansky/dlx/examples/langford@latest

## Usage

    langford [-n size] [-limit n] [-count] [-mirrors]

- `-n size` sets the largest number, 7 by default, from 1 to 64.
- `-limit n` sets how many arrangements to print, 1 by default. Zero prints all of them.
- `-count` enumerates every arrangement and prints the total instead, so `-limit` has no effect with it. The total grows quickly: n=12 took under a second on the machine used to write this, and n=15 took about three minutes.
- `-mirrors` counts an arrangement and its reversal as two.

`-help` prints the usage text to standard error.

Arrangements and totals go to standard output. Errors go to standard error, prefixed `langford:`, except for the flag package's own message about an invalid flag.

| Exit code | Meaning                                                 |
|-----------|---------------------------------------------------------|
| 0         | At least one arrangement exists, or `-help` was given   |
| 1         | There are no arrangements                               |
| 2         | Invalid usage, or a write failure                       |

## Examples

Find the first arrangement for n=7:

    langford

Output:

    1 7 1 2 5 6 2 3 4 7 5 3 6 4

List n=3 with its mirror image:

    langford -n 3 -mirrors -limit 0

Output:

    3 1 2 1 3 2
    2 3 1 2 1 3

Count the arrangements for n=7, then count them again with mirrors:

    langford -count
    langford -count -mirrors

Output:

    Arrangements: 26
    Arrangements: 52

The value n=5 has no arrangements, so this exits with code 1:

    langford -n 5

Output, on standard error:

    langford: no arrangement found
