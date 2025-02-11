# Parallel Testify Suite

## Motivation

The original testify suite [does not support](https://github.com/stretchr/testify/blob/master/README.md#suite-package)
parallel tests and has no plan to support it because it requires
backward-incompatible changes.

This package provides a `suite` implementation that supports parallel tests.
This is backward-incompatible with the testify suite. It also uses
generics to manage per-test data (and thus requires Go 1.18 or higher).

## Usage

See [suite_test.go](suite/suite_test.go) for example usage.
