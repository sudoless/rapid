# Copyright (c) 2026 SUDOLESS SRL
# SPDX-License-Identifier: BSD-3-Clause-Clear

LINT_CONFIG  := devz/golangci-lint.yaml
STRESS_COUNT ?= 200

.PHONY: help
help: ## list targets
	@grep -h -E '^[a-zA-Z0-9/_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: check
check: fmt-check lint test-race ## everything a PR must pass

.PHONY: fmt
fmt: ## format the Go sources with the config's formatters (gofumpt, goimports)
	@golangci-lint fmt --config=$(LINT_CONFIG) ./...

.PHONY: fmt-check
fmt-check: ## print the diff fmt would apply, and fail if there is one
	@golangci-lint fmt --config=$(LINT_CONFIG) --diff ./...

.PHONY: lint
lint: ## run golangci-lint over the module
	@golangci-lint run --config=$(LINT_CONFIG) ./...

.PHONY: lint-src
lint-src: ## run golangci-lint, skipping test files
	@golangci-lint run --config=$(LINT_CONFIG) --tests=false ./...

.PHONY: test
test: ## run the tests
	@go test ./...

.PHONY: test-race
test-race: ## run the tests with the race detector, shuffled, bypassing the cache
	@go test -race -count=1 -shuffle=on ./...

.PHONY: test-stress
test-stress: ## run the statistical and coverage checks on STRESS_COUNT random seeds
	@go test -tags stress -run '^TestStress$$' -count=$(STRESS_COUNT) ./...

.PHONY: bench
bench: ## run the benchmarks
	@go test -run='^$$' -bench=. -benchmem ./...
