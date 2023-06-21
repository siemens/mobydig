/*
Package dnsworker implements a simple limiting DNS client-request execution
pool. Mobydig uses [DnsPool] with a pool of “DNS workers” for A/AAAA lookups.
Please note that the A/AAAA queries for a single fqdn are not concurrent.

Usage

	dnsclnt := dns.Client{}
	workers := dnsworker.New(
	    context.Background(),
	    4,                    // number of parallel DNS connections and thus workers
	    dnsclnt,              // DNS client
	    "127.0.0.1:53",        // address of server/resolver
	)
	workers.ResolveName(
	    "foobar.example.org",
	    func(addrs []string, error){
	        // do something with addrs, unless there's an error reported
	    })
	workers.Submit(func(conn *dns.Conn){
	    // do something with the DNS connection
	})

# Acknowledgements

Under its hood, [DnsPool] leverages [gammazero/workerpool] as
the limiting goroutine pool.

[github.com/gammazero/workerpool]: https://github.com/gammazero/workerpool
*/
package dnsworker
