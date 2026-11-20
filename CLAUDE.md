# CLAUDE.md

The repo is public. Commit messages, PRs and docs describe API and behaviour, never decision history.

## Commands

```sh
go test -race ./...                 # full suite
go test -run='^$' -bench=. ./...    # benchmarks
gofmt -l .                          # prints nothing when formatted
go test -args -h                    # lists the -rapid.* flags
```

`-flaky` and `-flaky.ext` opt into the tests skipped as flaky.

## Invariants

- The module depends on nothing but the standard library.
- Licensing is MPL-2.0, file by file: a file that carries the MPL header keeps it, and a modified file stays MPL-2.0.
- Generation is deterministic per seed. The `Example*` outputs pin the exact values drawn, so a change to the bitstream, a generator, or the shrinker shows up there as a behaviour change, to be explained before the expected output is updated.
- A fail file (`testdata/rapid/<test>/*.fail`) replays a failure from its recorded bitstream and is ignored when its version differs from `rapidVersion` (`persist.go`).
- The `go` line in `go.mod` is the floor. Features from newer Go sit behind a `//go:build go1.N` / `!go1.N` file pair, as `synctest_enabled.go` and `synctest_disabled.go` do.

## Conventions

Tests sit beside their subject: `foo_test.go` (package `rapid`, internals), `foo_external_test.go` (package `rapid_test`, public API), `foo_example_test.go` (godoc examples).
