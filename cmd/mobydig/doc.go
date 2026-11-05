/*
Mobydig digs and verifies the DNS names of containers and services on all
networks attached to a specific container.

Usage:

	mobydig [flags] containername

The flags are:

	--debug
		enable debugging output.
	--indent uint
		indentation width in the range [0..80] (default 3).
	--spinner duration
		spinner interval, at least 10ms (default 100ms).
	--workers uint
		number of DNS and ping workers in the range [1..10] (default 5).
*/
package main
