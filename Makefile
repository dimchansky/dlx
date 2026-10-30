.PHONY: check task test test_race test_cover fmt vet lint fix tidy_check fuzz vulncheck

# Checks only; never modifies files. Same vet, lint and test passes as CI. CI's
# other checks stay separate: tidy_check needs Go 1.23+ and vulncheck Go 1.26+,
# and check also has to work with Go 1.19.
check: vet lint test_race test

# Rewrites files first, then checks; sub-makes keep the steps sequential under make -j.
task:
	@$(MAKE) fix
	@$(MAKE) fmt
	@$(MAKE) check

test:
	@go test ./...

# -short skips the example totals that take minutes under the race detector.
test_race:
	@go test -race -short -shuffle=on ./...

test_cover:
	@go test -cover -covermode=atomic ./...

# tested with golangci-lint v2.15.0 (same version as CI)
fmt:
	@golangci-lint fmt

vet:
	@go vet ./...

lint:
	@golangci-lint run

fix:
	@go fix ./...

tidy_check:
	@go mod tidy -diff

fuzz:
	@go test -run='^$$' -fuzz='^FuzzSolve$$' -fuzztime=30s .

# same version as CI; needs a Go release that govulncheck supports
vulncheck:
	@go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
