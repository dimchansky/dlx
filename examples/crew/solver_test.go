package main

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func readSample(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile("pairings.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// bruteForceCosts tries every subset of the candidates without the exact cover
// machinery, and returns the cheapest cost of each different partition of the
// flights, sorted. Candidates with equal flights give the same partition, which
// is how solve counts rosters.
func bruteForceCosts(p problem) []int {
	allFlights := 1<<len(p.flights) - 1
	cheapest := map[string]int{}

	var visit func(next, covered, cost int, chosen []int)
	visit = func(next, covered, cost int, chosen []int) {
		if next == len(p.pairings) {
			if covered != allFlights {
				return
			}
			sorted := append([]int{}, chosen...)
			sort.Ints(sorted)
			key := fmt.Sprint(sorted)
			if best, isSeen := cheapest[key]; !isSeen || cost < best {
				cheapest[key] = cost
			}
			return
		}
		visit(next+1, covered, cost, chosen)

		candidate := p.pairings[next]
		mask := 0
		for _, flight := range candidate.flights {
			mask |= 1 << flight
		}
		if covered&mask == 0 {
			visit(next+1, covered|mask, cost+candidate.cost, append(chosen, mask))
		}
	}
	visit(0, 0, 0, nil)

	costs := make([]int, 0, len(cheapest))
	for _, cost := range cheapest {
		costs = append(costs, cost)
	}
	sort.Ints(costs)
	return costs
}

func rosterCosts(rosters []roster) []int {
	costs := make([]int, len(rosters))
	for i, r := range rosters {
		costs[i] = r.cost
	}
	return costs
}

func checkRoster(t *testing.T, p problem, r roster) {
	t.Helper()

	covered := make([]int, len(p.flights))
	cost := 0
	for _, chosen := range r.pairings {
		cost += chosen.cost
		for _, flight := range chosen.flights {
			covered[flight]++
		}
	}
	if cost != r.cost {
		t.Errorf("roster cost is %d, its pairings add up to %d", r.cost, cost)
	}
	for flight, times := range covered {
		if times != 1 {
			t.Errorf("flight %s covered %d times", p.flights[flight], times)
		}
	}
}

func TestSolve_Sample(t *testing.T) {
	t.Parallel()

	p, err := parse(strings.NewReader(readSample(t)))
	if err != nil {
		t.Fatal(err)
	}

	rosters := solve(p)
	if len(rosters) != 40 {
		t.Errorf("found %d rosters, want 40", len(rosters))
	}
	if rosters[0].cost != 3730 {
		t.Errorf("cheapest roster costs %d, want 3730", rosters[0].cost)
	}
	for _, r := range rosters {
		checkRoster(t, p, r)
	}

	got, want := rosterCosts(rosters), bruteForceCosts(p)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("costs %v, brute force found %v", got, want)
	}
}

func TestSolve_RandomProblems(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewSource(1))
	solvable := 0
	for i := 0; i < 500; i++ {
		p := randomProblem(random)
		rosters := solve(p)
		if len(rosters) > 0 {
			solvable++
		}
		for _, r := range rosters {
			checkRoster(t, p, r)
		}
		got, want := rosterCosts(rosters), bruteForceCosts(p)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("problem %+v: costs %v, brute force found %v", p, got, want)
		}
	}
	if solvable == 0 {
		t.Error("no random problem had a roster, so the comparison proved little")
	}
}

// randomProblem makes a few flights and many short pairings, some with equal
// flights.
func randomProblem(random *rand.Rand) problem {
	const flightCount = 6
	p := problem{}
	for flight := 0; flight < flightCount; flight++ {
		p.flights = append(p.flights, fmt.Sprintf("F%d", flight))
	}
	pairingCount := 4 + random.Intn(10)
	for i := 0; i < pairingCount; i++ {
		flights := random.Perm(flightCount)[:1+random.Intn(3)]
		sort.Ints(flights)
		p.pairings = append(p.pairings, pairing{
			name:    fmt.Sprintf("P%d", i),
			cost:    random.Intn(20),
			flights: flights,
		})
	}
	return p
}

func TestSolve_SameFlightsKeepsCheapest(t *testing.T) {
	t.Parallel()

	p := problem{
		flights: []string{"F1", "F2"},
		pairings: []pairing{
			{name: "expensive", cost: 9, flights: []int{0, 1}},
			{name: "cheap", cost: 3, flights: []int{0, 1}},
			{name: "middle", cost: 5, flights: []int{0, 1}},
		},
	}

	rosters := solve(p)
	if len(rosters) != 1 {
		t.Fatalf("found %d rosters, want 1", len(rosters))
	}
	if got := rosters[0].pairings[0].name; got != "cheap" || rosters[0].cost != 3 {
		t.Errorf("roster uses %q at cost %d, want cheap at 3", got, rosters[0].cost)
	}
}

func TestSolve_UncoveredFlight(t *testing.T) {
	t.Parallel()

	p := problem{
		flights:  []string{"F1", "F2"},
		pairings: []pairing{{name: "only", cost: 1, flights: []int{0}}},
	}
	if rosters := solve(p); len(rosters) != 0 {
		t.Errorf("found %d rosters, want none", len(rosters))
	}
}
