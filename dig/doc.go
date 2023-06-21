/*
Package dig implements a DNS FQDN-to-address digger with optional (in)validation
of the network addresses dug out. The knack here is that the address digging and
validation is done from within a specific network namespace (of a Docker
container).

The digging and ping verification steps run concurrently, but under the
constraints of limited goroutines. That is, the maximum number of each worker
set is limited for DNS name-to-address resolution, as well as for address
validation using ICMP pings.

Digging and pinging is implemented in pure Go, leveraging the incredible Go
modules [miekg/dns] and [go-ping/ping].

# Notes

A good source for how Docker's/Moby's embedded DNS resolver works is
https://github.com/moby/libnetwork/blob/master/resolver.go.

[miekg/dns]: https://github.com/miekg/dns
[go-ping/ping]: https://github.com/go-ping/ping
*/
package dig
