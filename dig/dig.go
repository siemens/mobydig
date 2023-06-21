// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

import (
	"context"

	"github.com/siemens/mobydig/dnsworker"
	"github.com/siemens/mobydig/types"

	"github.com/miekg/dns"
)

// Digger digs the IPv4 and IPv6 addresses of FQDNs and then streams its
// findings over its “news” channel.
//
// By connecting the news (output) channel of a Verifier to the input channel of
// a Verifier the reachability of the addresses dug can automatically be
// verified by pinging them.
type Digger struct {
	workers *dnsworker.DnsPool
	news    chan types.NamedAddress
}

// New returns a new Digger with a maximum worker pool of the specified size as
// well as a “news stream”. This news channel sends NamedAddress elements as
// they are submitted for diggung, as well as the outcome(s) of the digs. Please
// note that the returned results channel is never closed by a Digger itself.
//
// I dunno what Sir Tim, Mick, Phil, and all the others might think of our
// digging here...
func New(size int, netnsref string) (*Digger, chan types.NamedAddress, error) {
	news := make(chan types.NamedAddress, size)
	dnsclnt := dns.Client{
		Net: "tcp", // ...since there's some chance that we need more than just two queries
	}
	workers, err := dnsworker.New(
		context.Background(), // ...pretty useless when using a pre-allocated UDP client.
		size,
		&dnsclnt, "127.0.0.11:53",
		dnsworker.InNetworkNamespace(netnsref)) // ...hammer the whale, but not too much ;)
	if err != nil {
		return nil, nil, err
	}
	return &Digger{
		workers: workers,
		news:    news,
	}, news, nil
}

// DigNetworks digs the IP addresses visible on a specific set of Docker
// networks. Intermediate and final results are getting sent to the channel
// returned beforehand by New.
func (d *Digger) DigNetworks(ctx context.Context, nets []DockerNetwork) {
	names := AllFQDNsOnAttachedNetworks(nets)
	d.DigFQDNs(ctx, names)
}

// DigFQDNs digs the given list of “host names” (whatever “host names” actually
// might mean). Intermediate and final results are getting sent to the channel
// returned beforehand by New.
func (d *Digger) DigFQDNs(ctx context.Context, names []string) {
	// Initially sent all unverified FQDNs to get the ball rolling so that the
	// consumer knows which FQDNs are going to be dug up next. Also submit the
	// DNS worker jobs to resolve the FQDNs...
	for _, name := range names {
		name := dns.Fqdn(name) // TODO: search list???
		// Initially inform the consumer of any FQDN that will undergo
		// resolution later; please note that ResolveName will enqueue
		// resolutions and thus not block. We only block if the consumer doesn't
		// consume our news ... and then only until the context gets cancelled.
		select {
		case d.news <- &types.NamedAddressValue{
			FQDN: name,
		}:
		case <-ctx.Done():
			return
		}
		d.workers.ResolveName(ctx, name, func(addrs []string, err error) {
			for _, addr := range addrs {
				// Avoid blocking enless in case of the context getting
				// cancelled.
				select {
				case d.news <- &types.NamedAddressValue{
					FQDN: name,
					QualifiedAddressValue: types.QualifiedAddressValue{
						Address: addr,
						Quality: types.Unverified,
					},
				}:
				case <-ctx.Done():
					return
				}
			}
		})
	}
}

// StopWait waits for all queued tasks to get processed and then finally closes
// the news channel.
func (d *Digger) StopWait() {
	d.workers.StopWait()
	close(d.news)
}
