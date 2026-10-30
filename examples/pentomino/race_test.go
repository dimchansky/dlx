//go:build race

package main

// raceEnabled reports whether the race detector is on. It slows these searches
// by far more than it could help: each one runs on a single goroutine.
const raceEnabled = true
