# Test

This directory contains tests and testing docs for `Knative Build`:

* [Unit tests](#running-unit-tests) currently reside in the codebase alongside the code they test
* [End-to-end tests](#running-end-to-end-tests), of which there are two types:
## Running unit tests

To run all unit tests:

```bash
go test ./...
```

_By default `go test` will not run [the e2e tests](#running-end-to-end-tests), which need [`-tags=e2e`](#running-end-to-end-tests) to be enabled._


## Running end to end tests

To run [the e2e tests](./e2e), you need to have a running environment that meets
[the e2e test environment requirements](#environment-requirements), and you need to specify the build tag `e2e`.

```bash
go test -v -tags=e2e -count=1 ./test/e2e/...
```

`-count=1` is the idiomatic way to bypass test caching, so that tests will always run.

### One test case

To run one e2e test case, e.g. TestSimpleBuild, use [the `-run` flag with `go test`](https://golang.org/cmd/go/#hdr-Testing_flags):

```bash
go test -v -tags=e2e -count=1 ./test/e2e/... -run=<regex>
```

### Environment requirements

These tests require [a running `Knative Build` cluster.](/DEVELOPMENT.md#getting-started)
