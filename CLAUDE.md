<!-- Copyright (c) 2026 SUDOLESS SRL -->
<!-- SPDX-License-Identifier: BSD-3-Clause-Clear -->

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
- Licensing is split by file. A file that holds or derives from upstream code is MPL-2.0 (`LICENSE-MPL`): it keeps its notices, and editing it leaves it MPL-2.0. A new file written from scratch is Clear BSD (`LICENSE`) and opens with:

  ```go
  // Copyright (c) <year> SUDOLESS SRL
  // SPDX-License-Identifier: BSD-3-Clause-Clear
  ```
- Generation is deterministic per seed. The `Example*` outputs pin the exact values drawn, so a change to the bitstream, a generator, or the shrinker shows up there as a behaviour change, to be explained before the expected output is updated.
- A fail file (`testdata/rapid/<test>/*.fail`) replays a failure from its recorded bitstream and is ignored when its version differs from `rapidVersion` (`persist.go`).
- Only the latest Go release is supported, and `go.mod` names it.

## Conventions

Tests sit beside their subject: `foo_test.go` (package `rapid`, internals), `foo_external_test.go` (package `rapid_test`, public API), `foo_example_test.go` (godoc examples).
