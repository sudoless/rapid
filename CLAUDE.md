<!-- Copyright (c) 2026 SUDOLESS SRL -->
<!-- SPDX-License-Identifier: BSD-3-Clause-Clear -->

# CLAUDE.md

The repo is public. Commit messages, PRs and docs describe API and behaviour, never decision history.

## Commands

```sh
make check     # fmt-check, lint, test-race: what a PR must pass
make fmt       # gofumpt and goimports, through golangci-lint
make help      # every target
```

Statistical tests are skipped unless `-flaky` (package `rapid`) or `-flaky.ext` (package `rapid_test`) is set; `make test-flaky` sets both.

## Invariants

- The module depends on nothing but the standard library.
- Licensing is split by file. A file that holds or derives from upstream code is MPL-2.0 (`LICENSE-MPL`): it keeps its notices, and editing it leaves it MPL-2.0. A new file written from scratch is Clear BSD (`LICENSE`) and opens with:

  ```go
  // Copyright (c) <year> SUDOLESS SRL
  // SPDX-License-Identifier: BSD-3-Clause-Clear
  ```
- Only the latest Go release is supported, and `go.mod` names it.

## Conventions

Tests sit beside their subject: `foo_test.go` (package `rapid`, internals), `foo_external_test.go` (package `rapid_test`, public API), `foo_example_test.go` (godoc examples).
