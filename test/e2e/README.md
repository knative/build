# End-to-End testing

## Running the tests

To run all e2e tests:
```
GOCACHE=off go test -tags e2e ./test/e2e/...
```

`GOCACHE=off` disables Go's test cache, so that tests results will not be
cached and the test will always run.

To run a single e2e test:

```
GOCACHE=off go test -tags e2e ./test/e2e/... -test.run=<regex>
```

## What the tests do

By default, tests use your current Kubernetes config to talk to your currently
configured cluster.

When they run tests ensure that a namespace named `build-tests` exists, then
starts running builds in it.
