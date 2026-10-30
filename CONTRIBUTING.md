# Contributing

## Before you send a change

Run the same checks as CI:

    make check

This needs [golangci-lint](https://golangci-lint.run/) v2.15.0, the version CI uses. `make task` also applies `go fix` and the formatters first.

Tests run with the race detector and `-short`, because the slowest example totals take minutes under it, and then once in full without it. `make fuzz` runs `FuzzSolve` for 30 seconds. If it writes a failing input to `testdata/fuzz`, commit that file with the fix.

## Conventions

- The module stays on Go 1.19 and has no dependencies. CI tests Go 1.19, oldstable and stable.
- A change to the `Matrix` contract updates the README and `doc.go`, and comes with a test.
- A new example needs a README that explains how the problem maps to exact cover.
- Commit subjects are short and imperative, and each commit does one thing.

## Releasing

Releases are tags on master, and the tag is the published version:

    git switch master && git pull
    git tag -a v1.2.3 -m v1.2.3
    git push origin v1.2.3

Pushing the tag starts the release workflow. It checks that the tag is a valid version and that its commit is on master, runs `go vet` and the tests, and then creates the GitHub Release with generated notes. A tag with a prerelease suffix, such as `v1.2.3-rc.1`, becomes a prerelease. The workflow never changes an existing release. It rejects v2 and later, which would need a `/vN` module path first.

Treat every version as permanent:

- Releases are immutable: once published, the tag is locked, and its name can't be used again even if the release is deleted.
- Once anyone fetches a version, the Go module proxy and checksum database keep it for good, so deleting or moving the tag doesn't take it back.
- To withdraw a bad version, publish a new one whose `go.mod` retracts it:

      // Solve reports wrong solutions when a row repeats a column.
      retract v1.2.3
