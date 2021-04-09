/*
Package testServer provides fasthttp testing server that's configured to respond
at very high RPS (capable of supporting up to 100k RPS).

This server is used when testing and benchmarking kiterunner to ensure that performance
does not degrade between feature enhancements and that expected behaviour does not change

The server is used for testing, and should not be used in a production environment.
We provide no support on how to use the testServer.
 */
package main
