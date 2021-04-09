/*
Package cmd provides all the commands for the kiterunner binary.

The commands are separated by file, with the prefix for each command corresponding to the parent command, i.e.
kitebuilderCompile.go corresponds to kb compile

there are a few global CLI flags that can be used to configure how kiterunner will operate. These are defined
by the globally exposed variables

The server can be started up across multiple ports to provide the simulation of scanning
multiple hosts

Usage

	go run ./cmd/testServer -p 14000-14500

 */
package cmd
