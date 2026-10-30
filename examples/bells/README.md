# Change ringing

A command-line search for extents of change ringing, written as an example of using package [dlx](https://github.com/dimchansky/dlx). It is the one example here where exact cover does only part of the job: the matrix finds candidates, and ordinary code throws almost all of them away. If exact cover is new to you, read the [introduction to the examples](../README.md) first.

## The problem

In English church towers, bells are rung in a changing order rather than as a tune. Each bell is number 1, 2, 3 or 4, from the lightest to the heaviest, and each one sounds once in every round of ringing. One round is called a **row**, and it is written as the order of the bells: `2134` means bell 2 sounds first, then 1, 3 and 4. Ringing starts from `1234`, which is called **rounds**.

A bell can weigh hundreds of kilograms and swings on a wheel, so a ringer can only make it sound a little earlier or later than last time. That gives the rule for going from one row to the next: **a bell may move by at most one place**. So two neighbouring bells can swap, and nothing bigger can happen.

An **extent** is the longest piece of ringing with no repeats. It follows three rules:

1. It starts and ends with rounds.
2. Every possible row comes exactly once, apart from rounds, which ends the piece as well.
3. From one row to the next, each bell moves by at most one place.

Four bells can stand in 4 x 3 x 2 x 1 = 24 orders, so an extent on 4 bells has 24 rows, and then rounds again. How many extents are there? That is the question this example answers.

### The changes

A step from one row to the next is called a **change**. A change is a rule about places, not about bells: "swap whoever is in places 1 and 2". On 4 bells, exactly four changes obey the rule. Places can only be swapped with a neighbouring place, and the only two swaps that fit in one step without overlapping are 1-2 and 3-4.

| Letter | What it does              | From `1234` |
|--------|---------------------------|-------------|
| a      | swap places 1 and 2       | `2134`      |
| b      | swap places 2 and 3       | `1324`      |
| c      | swap places 3 and 4       | `1243`      |
| d      | do a and c together       | `2143`      |

Doing the same change twice brings you back. So an arrow from a row to the next can always be turned around, and if you ring an extent backwards you get another extent.

### Three bells

Three bells are small enough to see the whole problem. There are 6 rows and two changes, a and b. Drawing a line between rows that are one change apart gives a ring:

![The six rows of three bells in a ring: 123, 213, 231, 321, 312, 132 and back to 123. Neighbouring rows are joined by change a or change b in turn.](ring.svg)

An extent has to walk through all six rows and come back, so it must go once around the ring. Starting at `123`, there are two ways to do that: clockwise and anticlockwise. They are the only two extents on 3 bells:

    123 213 231 321 312 132 123
    123 132 312 321 231 213 123

The rows of 4 bells make the same kind of picture, with 24 rows and 3 or 4 lines from each, but no ring to follow by eye. That is where a search comes in.

## How it becomes a matrix

Forget about the order for a moment and look at one row of bells in an extent. Exactly one step leaves it, and exactly one step arrives at it. That is the part exact cover can say.

**One column of the matrix is either "the step out of this row" or "the step into this row".** **One row of the matrix is one allowed step, from one row of bells to another.**

| Matrix part | Meaning                                  | 3 bells | 4 bells, a b c | 4 bells, a b c d |
|-------------|------------------------------------------|---------|----------------|------------------|
| Column      | The step out of a row of bells           | 6       | 24             | 24               |
| Column      | The step into a row of bells             | 6       | 24             | 24               |
| Row         | An allowed step                          | 12      | 72             | 96               |
| 1s in a row | The row it leaves and the row it reaches | 2       | 2              | 2                |

This is the whole matrix for 3 bells. Each step has one 1 under `out` and one 1 under `in`. Careful with the two meanings of "row": a table row is a step, and the labels such as 123 are rows of bells.

| Step | out 123 | out 132 | out 213 | out 231 | out 312 | out 321 | in 123 | in 132 | in 213 | in 231 | in 312 | in 321 |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| 123 to 213 (a) | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 |
| 123 to 132 (b) | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| 132 to 312 (a) | 0 | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 0 |
| 132 to 123 (b) | 0 | 1 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 0 |
| 213 to 123 (a) | 0 | 0 | 1 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 0 |
| 213 to 231 (b) | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 0 |
| 231 to 321 (a) | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 |
| 231 to 213 (b) | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 |
| 312 to 132 (a) | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| 312 to 321 (b) | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 1 |
| 321 to 231 (a) | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 1 | 0 | 0 |
| 321 to 312 (b) | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 1 | 0 |

For example, the first step in the table leaves `123`, so it has a 1 under `out 123`, and it reaches `213`, so it has a 1 under `in 213`. Picking both "123 to 213" and "123 to 132" is impossible, because both have a 1 under `out 123`. No slack rows are needed, because every row of bells really does have one step out and one step in.

### What the matrix misses

An exact cover picks one step out of every row of bells and one step into it. Follow the arrows from any row and you must come back to where you started, because no row can be entered twice. So an exact cover is a set of closed loops, with every row of bells in exactly one loop. This is called a **cycle cover**.

An extent is a cycle cover with **only one loop**. The matrix can't tell the difference. The 3 bell matrix above has four exact covers, and only two of them are extents:

![The four cycle covers of three bells. Covers 1 and 4 are three pairs of rows swapping back and forth. Covers 2 and 3 go once around the ring, one each way, and are the two extents.](covers.svg)

Covers 1 and 4 are three small loops: each pair of rows swaps back and forth, so the ringing never leaves its first two rows. Yet every row of bells has one step out and one step in, so these covers satisfy every column.

"Only one loop" isn't a rule about any one step, so it doesn't fit into a column. It depends on the whole set of steps. So the example does what you do with any rule that a matrix can't hold: it checks each solution afterwards, and rejects the ones that fail. The `AcceptSolution` callback is the place for this, as it is for the costs in the [crew example](../crew).

```mermaid
flowchart LR
    M["Matrix of steps"] --> S["dlx finds a cycle cover"]
    S --> F["Follow the arrows from rounds"]
    F --> Q{"Back at rounds after every row?"}
    Q -- "Yes" --> E["An extent"]
    Q -- "No, the loop closed early" --> R["Reject it"]
    E --> S
    R --> S
```

## How a solution decodes back

dlx reports column numbers. The rows of bells are numbered in sorted order, with rounds as number 0, and the `in` columns come after the `out` columns. So with n rows of bells, the column for "out of row i" is i, and the column for "into row i" is n + i.

For 3 bells, n is 6 and `213` is row number 2, so the step from `123` to `213` comes back as `[0 8]`: out of row 0, and into row 8 - 6 = 2. [solver.go](solver.go) reads every step into a list `next`, where `next[i]` is the row after row i. Then `extentOf` starts at rounds and follows `next`. If it returns to rounds before it has seen every row, the cover has more than one loop, and it's rejected.

A loop has no beginning, so each extent comes from exactly one cover, and the walk starts at rounds. The two directions of a loop are two different covers, so an extent and its reverse count separately. That matches the way ringers count them.

## Installation

    go install github.com/dimchansky/dlx/examples/bells@latest

## Usage

    bells [-bells 3|4] [-changes letters] [-limit n] [-count]

- `-bells n` sets the number of bells, 4 by default. Only 3 and 4 are accepted, for the reasons under [Limits](#limits).
- `-changes letters` lists the allowed changes, from the table above, in any order. The default is every single swap: `ab` on 3 bells and `abc` on 4 bells. `abcd` allows all four changes, and takes a few seconds with `-count`.
- `-limit n` sets how many extents to print, 1 by default. Zero prints all of them.
- `-count` goes through every cycle cover and prints the totals instead, so `-limit` has no effect with it.

`-help` prints the usage text to standard error.

Extents and totals go to standard output. Error messages go to standard error, and most start with `bells:`.

| Exit code | Meaning                                               |
|-----------|-------------------------------------------------------|
| 0         | At least one extent exists, or `-help` was given      |
| 1         | There is no extent with these changes                 |
| 2         | Invalid usage, or a write failure                     |

## Examples

Print both extents on 3 bells:

    bells -bells 3 -limit 0

Output:

    Extent 1:
    123
    213
    231
    321
    312
    132
    123
    Extent 2:
    123
    132
    312
    321
    231
    213
    123

Count the extents on 4 bells with every change allowed:

    bells -changes abcd -count

Output:

    Cycle covers: 2655552
    Extents: 10792

With changes a and b alone, the bell in place 4 never moves, so most rows can't be reached. This exits with code 1:

    bells -changes ab -count

Output:

    Cycle covers: 256
    Extents: 0

and on standard error:

    bells: no extent found

## Number of extents

There are 15 sets of changes on 4 bells. Four of them have extents:

| Changes | Cycle covers | Extents | Covers per extent, about |
|---------|--------------|---------|--------------------------|
| `abc`   | 28561        | 88      | 325                      |
| `abd`   | 16384        | 106     | 155                      |
| `bcd`   | 16384        | 106     | 155                      |
| `abcd`  | 2655552      | 10792   | 246                      |

The other 11 sets have none, because from rounds they can't reach every row. The tests check this for each of them.

Of the 10792 extents, half are the reverses of the other half, which leaves 5396 different loops.

One of the 106 extents that use `bcd` is the method that ringers call Plain Bob Minimus, which repeats the changes d b d b d b d c three times:

    1234 2143 2413 4231 4321 3412 3142 1324
    1342 3124 3214 2341 2431 4213 4123 1432
    1423 4132 4312 3421 3241 2314 2134 1243
    1234

## How the results are checked

- The total of 10792 is published in ["Ringing the changes"](https://plus.maths.org/content/ringing-changes) by Burkard Polster and Marty Ross, in Plus magazine, 2009. The article calls a row a "change". I found no published source for the other totals, including the 88, so they rest on the checks below.
- A separate walk over the rows finds every extent one step at a time, with no matrix and no cycle covers. The tests compare the two lists, extent by extent, for every set of changes.
- A separate count of the cycle covers goes through the rows in turn and keeps a tally of the choices so far. It involves no search, and it must give the same total.
- Both of those build their own rows and read the changes off pairs of rows, so they share no data with the solver.
- Each extent is checked against the three rules, and Plain Bob Minimus has to be among the extents.

`go test` checks everything. `go test` takes several seconds, nearly all of it on `abcd`. `go test -short` leaves out the comparisons for the five largest sets, though it still checks the total of 88. Use `-short` with `-race`.

## Limits

Most of the work here is not exact cover. For `abcd`, dlx produces 2655552 covers and the filter keeps 10792, so only about 1 cover in 246 is an extent. The matrix helps very little with the real difficulty of the problem, which is getting a single loop.

So this is a poor way to solve this problem, and it's here to show where exact cover stops. On the machine used to write this, `-changes abcd -count` takes about 3 seconds, while the plain walk in the tests finds the same 10792 extents in under a tenth of a second. The walk knows about loops as it goes, and leaves a dead end at once. dlx has to finish each cover before anyone looks at its loops.

It also won't grow. Changes a and c split the rows into groups of four, such as `12345`, `21345`, `12435` and `21435` on 5 bells. Each group can be covered in 4 ways on its own, whatever the other groups do, so the number of covers is at least 4 multiplied by itself once per group. The table for 4 bells shows exactly this: `ac` has 6 groups and 4096 covers.

| Bells | Rows | Groups of four | At least this many covers |
|-------|------|----------------|---------------------------|
| 4     | 24   | 6              | 4096                      |
| 5     | 120  | 30             | a 19 digit number         |
| 6     | 720  | 180            | a 109 digit number        |

At about a million covers a second, 5 bells would take tens of thousands of years, and that is only a lower limit. So the command stops at 4 bells.
