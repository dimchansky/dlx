package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// maxCost bounds a pairing's cost. Pairings in a roster cover different flights,
// so this also keeps the total cost of any roster well inside an int.
const maxCost = 1_000_000

// pairing is a sequence of flights one crew could work.
type pairing struct {
	name    string
	cost    int
	flights []int // indices into problem.flights, ascending
}

// problem is a day's flights and the candidate pairings that could cover them.
type problem struct {
	flights  []string
	pairings []pairing
}

// parse reads a problem. Blank lines and lines starting with '#' are skipped.
// The first other line is "flights" followed by the flight names, and each line
// after it is a pairing: a name, a cost, and the flights it covers.
func parse(r io.Reader) (problem, error) {
	p := problem{}
	flightIndex := map[string]int{}
	names := map[string]bool{}

	s := bufio.NewScanner(r)
	for line := 1; ; line++ {
		// Err comes first: a reader that fails mid-line still yields the partial
		// line, with Scan returning true and Err already set.
		scanned := s.Scan()
		if err := s.Err(); err != nil {
			return problem{}, fmt.Errorf("line %d: reading input: %w", line, err)
		}
		if !scanned {
			break
		}

		fields := strings.Fields(s.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}

		var err error
		if len(p.flights) == 0 {
			err = p.parseFlights(fields, flightIndex)
		} else {
			err = p.parsePairing(fields, flightIndex, names)
		}
		if err != nil {
			return problem{}, fmt.Errorf("line %d: %w", line, err)
		}
	}

	if len(p.flights) == 0 {
		return problem{}, errors.New("no flights line")
	}
	return p, nil
}

func (p *problem) parseFlights(fields []string, flightIndex map[string]int) error {
	if fields[0] != "flights" {
		return errors.New(`expected a line starting with "flights"`)
	}
	if len(fields) == 1 {
		return errors.New("expected at least one flight")
	}
	for _, flight := range fields[1:] {
		if _, isDuplicate := flightIndex[flight]; isDuplicate {
			return fmt.Errorf("flight %q listed twice", flight)
		}
		flightIndex[flight] = len(p.flights)
		p.flights = append(p.flights, flight)
	}
	return nil
}

func (p *problem) parsePairing(fields []string, flightIndex map[string]int, names map[string]bool) error {
	if len(fields) < 3 {
		return errors.New("expected a name, a cost and at least one flight")
	}
	name := fields[0]
	if names[name] {
		return fmt.Errorf("pairing %q listed twice", name)
	}
	names[name] = true

	cost, err := strconv.Atoi(fields[1])
	if err != nil || cost < 0 || cost > maxCost {
		return fmt.Errorf("invalid cost %q: want 0 to %d", fields[1], maxCost)
	}

	flights := make([]int, 0, len(fields)-2)
	seen := map[int]bool{}
	for _, flight := range fields[2:] {
		index, isKnown := flightIndex[flight]
		if !isKnown {
			return fmt.Errorf("unknown flight %q", flight)
		}
		if seen[index] {
			return fmt.Errorf("flight %q listed twice in pairing %q", flight, name)
		}
		seen[index] = true
		flights = append(flights, index)
	}
	sort.Ints(flights)

	p.pairings = append(p.pairings, pairing{name: name, cost: cost, flights: flights})
	return nil
}
