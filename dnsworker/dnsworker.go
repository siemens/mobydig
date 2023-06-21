// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dnsworker

import (
	"context"
	"fmt"
	"sync"

	"github.com/gammazero/workerpool"
	"github.com/miekg/dns"
	"github.com/thediveo/lxkns/ops"
	"github.com/thediveo/lxkns/ops/relations"
	"github.com/thediveo/lxkns/species"
)

// DnsPool is a (size-limited) pool of DNS client connections talking with the
// same DNS resolver address.
type DnsPool struct {
	netns   relations.Relation // network namespace to ping from, or nil.
	workers *workerpool.WorkerPool
	mu      sync.Mutex // protects the pool of DNS connections
	free    []*dns.Conn
}

// DnsPoolOption can be passed to New when creating new [DnsPool] objects.
type DnsPoolOption func(*DnsPool)

// New returns a pool of the specified size of DNS client connections, with each
// connection using the specified context and talking to the same DNS resolver
// address.
//
// DNS tasks are submitted using [DnsPool.Submit] in form of task functions
// receiving a concrete [dns.Conn].
//
// The passed context is used for creating (dialing) the DNS client connections
// only. It is not directly passed to the submitted DNS tasks, so task
// submitters are themselves responsible for capturing the necessary context in
// their task function closure.
//
// To operate a DnsPool in a network namespace different to that of the OS-level
// thread of the caller specify the [InNetworkNamespace] option and pass it a
// filesystem path that must reference a network namespace (such as
// "/proc/666/ns/net").
func New(ctx context.Context, size int, dnsclnt *dns.Client, addr string, options ...DnsPoolOption) (*DnsPool, error) {
	dnspool := &DnsPool{
		workers: workerpool.New(size),
	}
	for _, opt := range options {
		opt(dnspool)
	}
	// Create the DNS client connections for the workers ... please note that we
	// might
	free := make([]*dns.Conn, 0, size)
	dial := func() interface{} {
		for i := 0; i < size; i++ {
			conn, err := dnsclnt.DialContext(ctx, addr)
			if err != nil {
				// Immediately release all connections created so far.
				for _, conn := range free {
					conn.Close()
				}
				return err
			}
			free = append(free, conn)
		}
		return nil
	}
	// Dial the connections in the requested network namespace, if necessary.
	var err error
	var dialerr interface{}
	if dnspool.netns != nil {
		dialerr, err = ops.Execute(dial, dnspool.netns)
	} else {
		dialerr = dial()
	}
	if err != nil {
		return nil, err
	}
	if dialerr != nil {
		return nil, dialerr.(error)
	}
	dnspool.free = free
	return dnspool, nil
}

// InNetworkNamespace optionally runs a DnsPool inside the network namespace
// referenced by the specified filesystem path.
func InNetworkNamespace(netnsref string) DnsPoolOption {
	return func(p *DnsPool) {
		p.netns = ops.NewTypedNamespacePath(netnsref, species.CLONE_NEWNET)
	}
}

// Submit a task to the DNS client connection pool, where it gets enqueued to be
// executed on an available DNS client connection.
func (p *DnsPool) Submit(task func(conn *dns.Conn)) {
	p.workers.Submit(func() { p.task(task) })
}

// ResolveName is a convenience method for submitting A/AAAA queries and
// gathering the results. The results (resolved IP addresses in textual format)
// or an error if resolution failed is passed to the specified callback function
// fn.
//
// fn is called only once after completing both A and AAAA queries, so fn always
// gets to see all IP addresses from all IP families to see (if any).
//
// Please note that when the passed context is cancelled this will cancel all
// in-flight as well as scheduled name resolution jobs.
func (p *DnsPool) ResolveName(ctx context.Context, name string, fn func([]string, error)) {
	p.Submit(func(conn *dns.Conn) {
		var addrs []string
		var err error
		defer func() { fn(addrs, err) }() // ...ensure triggering the result callback on our way out

		dnsclnt := dns.Client{}
		nadanothing := true
		for _, addrType := range []uint16{dns.TypeA, dns.TypeAAAA} {
			// don't try to resolve the name if the context has been cancelled;
			// trigger the callback immediately with the context error.
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			default:
			}

			msg := dns.Msg{
				MsgHdr: dns.MsgHdr{Id: dns.Id()},
			}
			name := dns.Fqdn(name)
			msg.SetQuestion(name, addrType) // TODO: search list???
			var r *dns.Msg
			r, _, err = dnsclnt.ExchangeWithConn(&msg, conn)
			if err != nil {
				return
			}
			for _, rr := range r.Answer {
				if addrRR, ok := rr.(*dns.A); ok {
					nadanothing = false
					addrs = append(addrs, addrRR.A.String())
					continue
				}
				if addrRR, ok := rr.(*dns.AAAA); ok {
					nadanothing = false
					addrs = append(addrs, addrRR.AAAA.String())
				}
			}
		}
		// If we neither got A nor AAAA answers then we consider this to be an
		// error. This ensures to send an error to the callback together with
		// the nil list of resolved IP addresses.
		if nadanothing {
			err = fmt.Errorf("ResolveName: query for %q yields no answers", name)
		}
	})
}

// task grabs the next free DNS client and passes it to the specified function.
// After the function returns, the connection is put back into the free list.
func (p *DnsPool) task(task func(conn *dns.Conn)) {
	// pop off a free DNS client connection,
	// https://ueokande.github.io/go-slice-tricks/,
	p.mu.Lock()
	if len(p.free) == 0 {
		panic("no free DNS client connection available")
	}
	last := len(p.free) - 1
	conn := p.free[last]
	p.free = p.free[:last]
	p.mu.Unlock()
	// run the task with its assigned DNS client connection...
	task(conn)
	// ...and push the DNS client connection back into the free list.
	p.mu.Lock()
	p.free = append(p.free, conn)
	p.mu.Unlock()
}

// StopWait waits for all enqueued address lookup or generic DNS request tasks
// to finish, and then shuts down the pool.
func (p *DnsPool) StopWait() {
	p.workers.StopWait()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, conn := range p.free {
		conn.Close()
	}
	p.free = nil
}
