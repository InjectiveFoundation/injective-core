# Injective Interchain Tests

This directory contains tests for the Injective interchain module.

## Running Tests

To run the tests, execute the following command:

```bash
$ make image
$ cd interchaintest
$ go test -count=1 -v -timeout 6000s -run ^TestMempoolLanes$ github.com/InjectiveLabs/injective-core/interchaintest
```

Note that `-count=1` is used to prevent Go caching, timeout is increased to 6000s to allow for longer test runs.

You may also use:

```bash
$ export CONTAINER_LOG_TAIL=1000
$ export SHOW_CONTAINER_LOGS=always
```

To see the Docker container logs.
