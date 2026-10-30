package main

import (
	"fmt"
	"sort"

	"github.com/dimchansky/dlx"
)

// roster is a set of pairings that covers every flight exactly once.
type roster struct {
	pairings []pairing // ordered by first flight
	cost     int
}

// solve returns every roster, cheapest first. Rosters of equal cost stay in the
// order the search found them.
//
// Each flight is a column and each candidate pairing is a row that covers its
// flights, so an exact cover is a roster: every flight flown by exactly one crew.
//
// dlx has no costs, so finding the cheapest roster means enumerating them all and
// comparing totals here. dlx also reports a row by its columns alone, which means
// two pairings with the same flights can't be told apart. Only the cheaper one can
// be part of the cheapest roster, so each set of flights gets one matrix row, and
// the map remembers the cheapest pairing for that set. Rows keep their AddRow
// column order, so a row from a solution gives the same key as the pairing it
// came from.
func solve(p problem) []roster {
	m := dlx.NewMatrix(len(p.flights))
	cheapest := map[string]pairing{}
	for _, candidate := range p.pairings {
		key := fmt.Sprint(candidate.flights)
		best, isSeen := cheapest[key]
		if !isSeen {
			m.AddRow(candidate.flights...)
		}
		if !isSeen || candidate.cost < best.cost {
			cheapest[key] = candidate
		}
	}

	rosters := []roster{}
	m.Solve(dlx.SolutionAccepterFunc(func(exactCover [][]int) bool {
		r := roster{pairings: make([]pairing, 0, len(exactCover))}
		for _, row := range exactCover {
			chosen := cheapest[fmt.Sprint(row)]
			r.pairings = append(r.pairings, chosen)
			r.cost += chosen.cost
		}
		sort.Slice(r.pairings, func(i, j int) bool {
			return r.pairings[i].flights[0] < r.pairings[j].flights[0]
		})
		rosters = append(rosters, r)
		return false
	}))

	sort.SliceStable(rosters, func(i, j int) bool { return rosters[i].cost < rosters[j].cost })
	return rosters
}
