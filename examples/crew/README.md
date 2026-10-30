# Crew pairing

A command-line solver for a small crew pairing problem, written as an example of using package [dlx](https://github.com/dimchansky/dlx). Airlines group the flights of a day into pairings, each a sequence of flights one crew can work. The task is to pick pairings so that every flight is flown by exactly one crew, at the lowest cost. This kind of problem is called set partitioning, and it appears wherever every job must be done exactly once: delivery routes, shifts, machine schedules.

## How it maps to exact cover

The mapping is direct, and it's the simplest one here:

| Matrix part | Meaning                                                   | Count in pairings.txt  |
|-------------|-----------------------------------------------------------|------------------------|
| Column      | A flight, which must be flown by exactly one crew         | 12                     |
| Row         | A candidate pairing, covering its flights                 | 19, from 20 candidates |

The interesting part is what dlx leaves out, because real problems run into both of these:

- **dlx has no costs.** It finds solutions and doesn't rank them. So [solver.go](solver.go) enumerates every roster, adds up each one's cost, and sorts them. That is exhaustive, which is fine for a few dozen candidates and useless for a real airline's thousands. Real crew pairing uses integer programming.
- **dlx doesn't identify rows.** A solution lists each row's columns, not which row it was, so two pairings with the same flights look identical. Here that happens for A2 and D1 in [pairings.txt](pairings.txt). The solver gives each set of flights a single matrix row and remembers the cheapest pairing for it, since a dearer one can't be in the cheapest roster. As a result, the rosters are counted by how they divide up the flights, and a dearer duplicate never appears, even with `-limit 0`.

The number of solutions depends on the data. A flight that no pairing covers leaves its column empty, so there are no rosters at all, and that is reported as such.

The tests compare the cost of every roster with a brute-force search over subsets of the candidates, for the sample file and for 500 random problems.

## Installation

    go install github.com/dimchansky/dlx/examples/crew@latest

## Input

```
# Comments start with a hash.
flights F101 F102 F103
A1  900 F101 F102
A2  700 F103
```

- Blank lines and lines starting with `#` are skipped. A `#` later in a line is not a comment.
- The first other line is the word `flights` followed by every flight name.
- Each later line is a pairing: a unique name, a cost from 0 to 1000000, and the flights it covers. Fields are separated by whitespace.
- Naming an unlisted flight is an error, which catches typos.
- Input is read to the end, so if you type at a terminal, finish with Ctrl-D.

[pairings.txt](pairings.txt) is a made-up sample with 12 flights.

## Usage

    crew [-limit n] < pairings.txt

`-limit n` sets how many rosters to print, cheapest first, 1 by default. Zero prints all of them. All rosters are found and held in memory whatever the limit. Rosters of equal cost appear in the order the search found them.

`-help` prints the usage text to standard error.

Rosters go to standard output. Errors go to standard error, prefixed `crew:`, except for the flag package's own message about an invalid flag.

| Exit code | Meaning                                                    |
|-----------|------------------------------------------------------------|
| 0         | At least one roster exists, or `-help` was given           |
| 1         | No roster covers every flight                              |
| 2         | Invalid usage, invalid input, or a read or write failure   |

## Examples

Run this from this directory. Find the two cheapest rosters:

    crew -limit 2 < pairings.txt

Output:

    Rosters covering every flight: 40

    Roster 1, cost 3730:
      B1    1150  F101 F102 F103 F104
      B5    1400  F105 F106 F107 F108
      B4    1180  F109 F110 F111 F112

    Roster 2, cost 3780:
      B1    1150  F101 F102 F103 F104
      A3     800  F105 F106
      A4     650  F107 F108
      B4    1180  F109 F110 F111 F112

Each line of a roster shows the pairing's name, its cost and its flights.
