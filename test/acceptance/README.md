# Acceptance tests suite #

Acceptance tests are similar to end-to-end (e2e) tests in that they test the entire system from the user's perspective.

Contrary to e2e tests they prefer to start within a single process and do not use disk persistence, so that costly network and disk I/O are prevented.
They should run as fast as possible, taking the risk that some errors in the system would only be caught by the full e2e suite.

* acceptance_test.go starts the `ginkgo` test suite

* simple_transfer.go - exercise `sendTransaction` and `callMethod` APIs

* consensus.go - exercise the internal consensus algo

