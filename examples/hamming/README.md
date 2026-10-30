# Perfect codes and the hat game

A command-line search for perfect codes, written as an example of using package [dlx](https://github.com/dimchansky/dlx). A perfect code answers a puzzle that looks impossible: seven prisoners win a guessing game 7 times out of 8, although each guess on its own is a coin flip. If exact cover is new to you, read the [introduction to the examples](../README.md) first.

## The puzzle

Seven prisoners stand in a room. Each is given a white or a black hat, chosen by a fair coin flip. A prisoner sees the other six hats but not their own. They may agree on a plan beforehand, but once the hats are on they can't communicate.

Then all of them answer at the same moment. Each one says "white", says "black", or passes. The team goes free if **at least one prisoner guesses, and every guess is right**. One wrong guess loses, and so does everybody passing.

An easy plan is that one prisoner guesses and the rest pass, which wins half the time. It seems impossible to do better, because the other hats say nothing about your own, so any single guess is right half the time. Yet the team can win 112 of the 128 possible patterns of hats, which is 7/8.

### Three prisoners

Three prisoners already show the trick. The plan: **if the two hats you see match, say the other colour. If they differ, pass.** There are 8 patterns of hats, each equally likely:

| Hats (W white, B black) | Prisoner 1 | Prisoner 2 | Prisoner 3 | Team |
|-------------------------|------------|------------|------------|------|
| W W W                   | B, wrong   | B, wrong   | B, wrong   | loses |
| W W B                   | pass       | pass       | B, right   | wins |
| W B W                   | pass       | B, right   | pass       | wins |
| W B B                   | W, right   | pass       | pass       | wins |
| B W W                   | B, right   | pass       | pass       | wins |
| B W B                   | pass       | W, right   | pass       | wins |
| B B W                   | pass       | pass       | W, right   | wins |
| B B B                   | W, wrong   | W, wrong   | W, wrong   | loses |

The team wins 6 of 8 patterns, which is 3/4, the best any plan can do for three prisoners. Count the guesses: 6 are right and 6 are wrong, so each guess really is a coin flip. The plan doesn't make anyone guess better. It **puts the wrong guesses together**: all 6 land on the 2 losing patterns, while each winning pattern has a single right guess and nothing else.

### Why 7/8 can't be beaten

Take any plan and any prisoner, and pair up the hat patterns that differ only in that prisoner's hat. The prisoner sees the same thing in both patterns of a pair, so they answer the same way in both. A guess is therefore right in one pattern of the pair and wrong in the other. Adding up over all prisoners, **every plan has as many right guesses as wrong guesses**.

A won pattern holds at least one right guess and no wrong one. A lost pattern holds at most 7 wrong guesses. So the wins can't exceed the right guesses, which equal the wrong guesses, which can't exceed 7 times the losses. With 128 patterns, wins of at most 7 times the losses means at least 16 losses, and at most 112 wins.

To reach that limit, every lost pattern must hold 7 wrong guesses and every won pattern exactly one right guess. Finding 16 patterns to lose on is what a perfect code does.

## From hats to words

Write a white hat as 0 and a black hat as 1. A pattern of hats is now a **word** such as `0110100`. Here are the only ideas needed:

- The **distance** between two words is the number of places in which they differ. `0110100` and `0110101` have distance 1, and `000` and `111` have distance 3.
- The **neighbourhood** of a word is the word itself plus every word at distance 1. A word of 7 digits has 7 neighbours, because any of its 7 digits can flip, so its neighbourhood has 8 words.
- A **perfect code** is a set of words, called code words, whose neighbourhoods contain every word exactly once. Every word is either a code word or one flip away from exactly one code word.

For words of 3 digits, draw the 8 words as the corners of a cube, so that a line joins two words at distance 1. The code `000`, `111` splits the corners into two neighbourhoods, one around each filled corner. Every corner is in exactly one of them.

![The eight words of three digits as corners of a cube. 000 with its neighbours 001, 010 and 100 form one group, and 111 with its neighbours 110, 101 and 011 form the other.](neighbourhoods.svg)

A cube of 7 digits can't be drawn, but the arithmetic is the same: there are 128 words, each neighbourhood holds 8, so a perfect code has 128 / 8 = 16 words.

### The plan, in terms of the code

The prisoners agree on a perfect code. Each prisoner then asks the same question: "With the hats I see, could my hat colour complete a code word?"

```mermaid
flowchart TD
    S["Try both colours in place of my own hat"] --> Q{"Does one colour make a code word?"}
    Q -- "Yes" --> G["Say the other colour"]
    Q -- "No" --> P["Pass"]
```

Both colours can't make a code word, because those two words would be at distance 1, and code words are at least 3 apart. Then there are two cases:

- **The hats are not a code word.** They are one flip away from exactly one code word, so exactly one prisoner can complete a code word, namely the one whose hat is the flipped digit. That prisoner says the other colour, which is the true one. Everyone else passes. The team wins.
- **The hats are a code word.** Every prisoner sees a code word that their own hat completes, so all 7 say the other colour, and all are wrong. The team loses.

So the team loses exactly on the 16 code words, and wins on the other 112. That is the limit proved above. The plan for three prisoners is this same plan with the code `000`, `111`.

## Which lengths have a perfect code

The neighbourhoods share out all the words evenly, so the neighbourhood size must divide the number of words. With words of length n, that means n+1 must divide 2 multiplied by itself n times, which works only when n+1 is itself a power of 2.

| Length n | Words | Neighbourhood size | Divides evenly? |
|----------|-------|--------------------|-----------------|
| 1        | 2     | 2                  | yes             |
| 3        | 8     | 4                  | yes             |
| 4        | 16    | 5                  | no              |
| 7        | 128   | 8                  | yes             |
| 8        | 256   | 9                  | no              |
| 15       | 32768 | 16                 | yes             |

The command checks this first, so it answers lengths such as 8 at once. Without the check, the search would have to prove the same thing the long way.

## How it becomes a matrix

**One column of the matrix is one word, which must lie in exactly one neighbourhood.** **One row of the matrix is one word offered as a code word, and it has a 1 in every column of that word's neighbourhood.**

| Matrix part | Meaning                                   | Count for length 3 | Count for length 7 |
|-------------|-------------------------------------------|--------------------|--------------------|
| Column      | A word to be covered                      | 8                  | 128                |
| Row         | A candidate code word                     | 8                  | 128                |
| 1s in a row | The candidate and its neighbours          | 4                  | 8                  |

This is the whole matrix for length 3. For example, row `001` has its 1s under `001` itself and under its neighbours `000`, `011` and `101`.

|     | 000 | 001 | 010 | 011 | 100 | 101 | 110 | 111 |
|-----|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| 000 | 1 | 1 | 1 | 0 | 1 | 0 | 0 | 0 |
| 001 | 1 | 1 | 0 | 1 | 0 | 1 | 0 | 0 |
| 010 | 1 | 0 | 1 | 1 | 0 | 0 | 1 | 0 |
| 011 | 0 | 1 | 1 | 1 | 0 | 0 | 0 | 1 |
| 100 | 1 | 0 | 0 | 0 | 1 | 1 | 1 | 0 |
| 101 | 0 | 1 | 0 | 0 | 1 | 1 | 0 | 1 |
| 110 | 0 | 0 | 1 | 0 | 1 | 0 | 1 | 1 |
| 111 | 0 | 0 | 0 | 1 | 0 | 1 | 1 | 1 |

Rows `000` and `111` together have exactly one 1 in every column, so `000`, `111` is an exact cover. This matrix has four exact covers, and each is a pair of opposite corners of the cube: `000`, `111`, then `001`, `110`, then `010`, `101`, then `011`, `100`.

Two rows can't both be picked if their neighbourhoods overlap, because some column would then have two 1s. That is what keeps code words 3 apart, and it needs no separate rule. No slack rows are needed either, because every word must be covered exactly once.

## How a solution decodes back

dlx reports column numbers, so each word needs a number. A word is read as a binary number: `101` is 4 + 0 + 1 = 5, and `0000111` is 4 + 2 + 1 = 7.

[solver.go](solver.go) adds each row with the candidate's own column first. So the first number of each reported row is a code word, and that is all the decoder reads. For length 7, the row for `0000111` comes back as `[7 6 5 3 15 23 39 71]`: the word 7, then its seven neighbours. The decoder keeps the 7 and sorts the 16 code words into order.

## Bonus: a football pool

The same matrix works with three digit values. In a football pool you predict several matches, each ending in a home win, a draw or an away win. Write these as 0, 1 and 2. How few coupons guarantee that one of them has at most one match wrong, whatever happens?

With 4 matches there are 3 × 3 × 3 × 3 = 81 possible results. A digit can change to two other values, so a word has 2 × 4 = 8 neighbours and a neighbourhood of 9 words. The neighbourhood of `0000` is itself and `1000`, `2000`, `0100`, `0200`, `0010`, `0020`, `0001`, `0002`. Since 81 / 9 = 9, a perfect code has 9 words, and those are the 9 coupons. The matrix has 81 columns and 81 rows, and the columns are numbered by reading words in base 3. Fewer coupons can't work, because 8 neighbourhoods reach at most 72 of the 81 results.

    hamming -base 3 -n 4

Output:

    Code 1:
    0000
    0111
    0222
    1012
    1120
    1201
    2021
    2102
    2210

If the real result is `2222`, the coupon `0222` has one match wrong. Other results work the same way, and each is within one match of exactly one coupon.

## Installation

    go install github.com/dimchansky/dlx/examples/hamming@latest

## Usage

    hamming [-base 2|3] [-n length] [-limit n] [-count]
    hamming [-n length] -hat
    hamming [-n length] -play hats

- `-base b` sets how many digit values a word uses, 2 by default. It can be 2 or 3.
- `-n length` sets the word length, 7 by default, from 1 to 14 in base 2 and from 1 to 8 in base 3. The limits are explained under [Limits](#limits).
- `-limit n` sets how many codes to print, 1 by default. Zero prints all of them.
- `-count` goes through every code and prints the total instead, so `-limit` has no effect with it.
- `-hat` prints the first code, then scores its plan against every pattern of hats. It works in base 2 only and ignores `-limit`.
- `-play hats` prints the first code, then one round against the given hats, such as `0110100`, where 0 is white and 1 is black. It works in base 2 only, takes as many digits as the word length, and ignores `-limit`.

`-count`, `-hat` and `-play` can't be combined. `-help` prints the usage text to standard error.

Results go to standard output. Error messages go to standard error and start with `hamming:`.

| Exit code | Meaning                                       |
|-----------|-----------------------------------------------|
| 0         | A code exists, or `-help` was given           |
| 1         | No perfect code exists for this base and length |
| 2         | Invalid usage, or a write failure             |

## Examples

Find the first code of length 7, the one the prisoners use:

    hamming

Output:

    Code 1:
    0000000
    0000111
    0011001
    0011110
    0101010
    0101101
    0110011
    0110100
    1001011
    1001100
    1010010
    1010101
    1100001
    1100110
    1111000
    1111111

Score its plan on every pattern of hats:

    hamming -hat

Output, after the same code:

    Hat patterns: 128
    Team wins:    112
    Team loses:   16
    Right guesses: 112
    Wrong guesses: 112

Play a round. `0110101` is not a code word, but it is one flip from the code word `0110100`, so the last prisoner alone speaks:

    hamming -play 0110101

Output, after the code:

    Hats: 0110101
    Prisoner 1 sees ?110101 and passes
    Prisoner 2 sees 0?10101 and passes
    Prisoner 3 sees 01?0101 and passes
    Prisoner 4 sees 011?101 and passes
    Prisoner 5 sees 0110?01 and passes
    Prisoner 6 sees 01101?1 and passes
    Prisoner 7 sees 011010? and says 1: right
    Team wins

And `0000111` is a code word, so everybody guesses wrong:

    hamming -play 0000111

Output, after the code:

    Hats: 0000111
    Prisoner 1 sees ?000111 and says 1: wrong
    Prisoner 2 sees 0?00111 and says 1: wrong
    Prisoner 3 sees 00?0111 and says 1: wrong
    Prisoner 4 sees 000?111 and says 1: wrong
    Prisoner 5 sees 0000?11 and says 0: wrong
    Prisoner 6 sees 00001?1 and says 0: wrong
    Prisoner 7 sees 000011? and says 0: wrong
    Team loses

Count the codes of length 7:

    hamming -count

Output:

    Codes: 240

Length 8 has no perfect code, so this exits with code 1:

    hamming -n 8

Output, on standard error:

    hamming: no perfect code found

## Number of codes

| Base | Length | Words | Words in a code | Perfect codes |
|------|--------|-------|-----------------|---------------|
| 2    | 1      | 2     | 1               | 2             |
| 2    | 3      | 8     | 2               | 4             |
| 2    | 7      | 128   | 16              | 240           |
| 3    | 1      | 3     | 1               | 3             |
| 3    | 4      | 81    | 9               | 72            |

Every other length the command accepts has none. Any of the 240 codes gives the same score: 112 wins.

## How the results are checked

- **Every code is checked against the definition.** The tests count, for each word, the code words at distance 0 or 1, without using the matrix.
- **The two totals are rebuilt with no search.** There is an old recipe for these codes that needs no solver. For length 7, give the seven places the labels 1 to 7 in some order, and group the words by the sum of the labels of their 1 digits, using binary addition without carrying. Trying all 5040 orders gives exactly 240 different codes, the same 240 the search finds. A similar recipe gives the same 72 for the football pool.
- **Nothing is missing.** A separate search tries sets of words at least 3 apart, with no matrix and no neighbourhoods, and finds 30 codes that contain `0000000` and 8 that contain `0000`. Shifting all words of a code by a fixed word gives another code, so every word lies in equally many codes. That gives 30 × 128 / 16 = 240, and 8 × 81 / 9 = 72.
- **The hat game.** The tests play all 128 patterns with each of the 240 codes and get 112 wins, 16 losses, 112 right guesses and 112 wrong guesses every time. They check that no prisoner's answer ever depends on their own hat. For three prisoners, they try all 531441 possible plans, with no code involved, and the best one wins 6 of 8 patterns, as the plan here does.
- **The shortcut for impossible lengths.** The tests confirm that the matrix itself has no exact cover for lengths 2, 4, 5 and 6 in base 2, and for lengths 2, 3 and 5 in base 3.

`go test` runs everything. `go test -short` skips the two slowest checks. Use `-short` with `-race`.

## Limits

The search doesn't reach length 15. A perfect code of length 15 exists, with 2048 words, and the tests build one directly with the recipe above. But dlx sees only a matrix of 32768 rows and columns, and on the machine used to write this it found no code in five minutes. So the command stops at 14 digits in base 2 rather than appearing to hang, and at 8 digits in base 3, the longest that stays within the same 16384 words. Length 7 is no trouble: the total takes about a millisecond.

The game needs a perfect code, so `-hat` and `-play` only work for 1, 3 or 7 prisoners. Other team sizes exit with code 1, and this example has no plan for them.

The 7/8 is the chance for the team, with hats chosen by fair coin flips. It doesn't change the 1/2 chance of any single guess.

## Sources

The hat game and the 7/8 result are described in the Wikipedia article [Hat puzzle](https://en.wikipedia.org/wiki/Hat_puzzle), and the football pool problem, with 9 coupons for 4 matches, in [Covering code](https://en.wikipedia.org/wiki/Covering_code).
