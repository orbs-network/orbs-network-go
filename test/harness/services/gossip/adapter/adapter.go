/*
Package adapter provides an in-memory implementation of the Gossip Transport adapter, meant for usage in fast tests that
should not use the TCP-based adapter, such as acceptance tests or sociable unit tests. It could also be used in unit tests
where mocking the adapter would make no sense (if, for instance, the real behavior of the adapter is required).
*/
package adapter
