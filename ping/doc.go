/*
Package ping implements an ICMP(v4/v6)-based IP address (in)validator.

[Pinger] objects support concurrent IP address validation jobs with maximum
goroutine limits. Individual Ping verdicts are streamed as they are decided, to
a channel returned when creating a new Pinger object. Here, a [QualifiedAddress]
consists of (at least) an IP address as well as the [types.Quality] state,
notably [types.Valid] and Invalid, but also [types.Verifying] and (initially)
[types.Unverified].

	         +---+
	string-->| P +-->ch QualifiedAddress
	         +---+

⚠ Please note that a [Pinger] initially emits any newly submitted address before
it undergoes verification (with its quality set to “verifying”), as well as
later the final verdict. The rationale is that especially interactive clients
can more easily manage their display so that all enqueued verifications are
early visible.

If needed, a Pinger can read from the addresses it has to verify from an input
channel until this input channel is closed.

	            +---+
	ch string-->| P +-->ch QualifiedAddress
	            +---+

Pingers can be operated in a pipeline in that they read the addresses to be
validated from an input channel and then stream the results to the Pinger's
output channel.

# Acknowledgements

Under its hood, [Pinger] leverages [gammazero/workerpool] as the limiting
goroutine pool.

[gammazero/workerpool]: https://github.com/gammazero/workerpool
*/
package ping
