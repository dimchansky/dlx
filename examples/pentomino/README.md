# Pentomino tiling

A command-line solver that tiles a rectangle with the 12 pentominoes, each used exactly once, written as an example of using package [dlx](https://github.com/dimchansky/dlx). A pentomino is a shape made of five squares joined edge to edge. Pieces may be rotated and flipped. Twelve pieces have 60 squares in all, so the board must have 60 cells.

## How it maps to exact cover

This is the classic geometric use of exact cover. Two different kinds of requirement share one matrix:

| Matrix part | Meaning                                                       | Count for 6x10 |
|-------------|---------------------------------------------------------------|----------------|
| Column      | A piece, which must be used exactly once                      | 12             |
| Column      | A board cell, which must be filled exactly once               | 60             |
| Row         | One piece, in one orientation, at one position on the board   | 2056           |

Each row has six 1s: its piece and the five cells it occupies. The piece columns matter: with cell columns alone, a tiling could use one piece three times and skip another.

[solver.go](solver.go) generates the distinct orientations of each piece, between 1 for the X and 8 for the asymmetric pieces, 63 in all. A solution's rows give the piece and cells directly, because dlx keeps each row's columns in `AddRow` order.

## Counting duplicates

Every tiling is reported together with its images under the board's symmetries, which are the two mirror flips and the half turn. Each one is a different exact cover, so `Tilings` counts all of them. `Distinct` counts each group once.

Here the matrix can't easily remove them, so the program compares each tiling with its images and counts it only if it is the smallest. This works because each group has exactly four members: the F piece has no symmetry and appears once, so no flip can give the same tiling, and a 60-cell board is never square, so quarter turns don't apply. The [Langford example](../langford) shows the other approach, which removes mirror images from the matrix itself.

| Board | Tilings | Distinct |
|-------|---------|----------|
| 6x10  | 9356    | 2339     |
| 5x12  | 4040    | 1010     |
| 4x15  | 1472    | 368      |
| 3x20  | 8       | 2        |
| 2x30  | 0       | 0        |
| 1x60  | 0       | 0        |

The tests check these totals against an independent backtracking search with its own table of pieces.

## Installation

    go install github.com/dimchansky/dlx/examples/pentomino@latest

## Usage

    pentomino [-size RxC] [-limit n] [-count]

- `-size RxC` sets the board as rows and columns, 6x10 by default. Any size with 60 cells is accepted.
- `-limit n` sets how many tilings to print, 1 by default. Zero prints all of them.
- `-count` enumerates every tiling and prints the totals instead, so `-limit` has no effect with it. This took about 4 seconds for 6x10 on the machine used to write this, while finding the first tiling took milliseconds.

`-help` prints the usage text to standard error.

Tilings and totals go to standard output. Errors go to standard error, prefixed `pentomino:`, except for the flag package's own message about an invalid flag.

| Exit code | Meaning                                                 |
|-----------|---------------------------------------------------------|
| 0         | At least one tiling exists, or `-help` was given        |
| 1         | The board has no tilings                                |
| 2         | Invalid usage, or a write failure                       |

## Examples

Find the first tiling of the default board. Each letter is the piece that covers that cell:

    pentomino

Output:

    Tiling 1:
    LLXFFTTTWW
    LXXXFFTWWN
    LUXUFZTWNN
    LUUUYZZZNV
    PPPYYYYZNV
    PPIIIIIVVV

Find the first tiling of a long, thin board:

    pentomino -size 3x20

Output:

    Tiling 1:
    UUXIIIIINNNFTWYYYYZV
    UXXXPPLNNFFFTWWYZZZV
    UUXPPPLLLLFTTTWWZVVV

Count the tilings of a 5x12 board:

    pentomino -size 5x12 -count

Output:

    Tilings: 4040
    Distinct: 1010

## Tests

`go test` runs every total. With `-short` or `-race` it skips the 4x15, 5x12 and 6x10 enumerations: the race detector makes them take minutes, and each search runs on a single goroutine.
