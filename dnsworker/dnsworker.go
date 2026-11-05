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
	netns   relations.Relation // network namespace to resolve names in, or nil.
	dnsclnt *dns.Client
	addr    string
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
// The passed context is used for creating (dialing) the initial DNS client
// connections only. It is not directly passed to the submitted DNS tasks, so
// task submitters are themselves responsible for capturing the necessary
// context in their task function closure.
//
// Connections that fail during name resolution get replaced by newly dialed
// connections.
//
// To operate a DnsPool in a network namespace different to that of the OS-level
// thread of the caller specify the [InNetworkNamespace] option and pass it a
// filesystem path that must reference a network namespace (such as
// "/proc/666/ns/net").
func New(ctx context.Context, size int, dnsclnt *dns.Client, addr string, options ...DnsPoolOption) (*DnsPool, error) {
	if size < 1 {
		return nil, fmt.Errorf("DnsPool: size must be at least 1, got: %d", size)
	}
	dnspool := &DnsPool{
		dnsclnt: dnsclnt,
		addr:    addr,
	}
	for _, opt := range options {
		opt(dnspool)
	}
	free := make([]*dns.Conn, 0, size)
	dialerr, err := inNetns(dnspool.netns, func() error {
		for range size {
			conn, err := dnsclnt.DialContext(ctx, addr)
			if err != nil {
				// Immediately release all connections created so far.
				for _, conn := range free {
					_ = conn.Close()
				}
				return err
			}
			free = append(free, conn)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if dialerr != nil {
		return nil, dialerr
	}
	dnspool.free = free
	dnspool.workers = workerpool.New(size)
	return dnspool, nil
}

// InNetworkNamespace optionally runs a DnsPool inside the network namespace
// referenced by the specified filesystem path. An empty path leaves the DnsPool
// in the network namespace of the caller.
func InNetworkNamespace(netnsref string) DnsPoolOption {
	return func(p *DnsPool) {
		if netnsref == "" {
			p.netns = nil
			return
		}
		p.netns = ops.NewTypedNamespacePath(netnsref, species.CLONE_NEWNET)
	}
}

// inNetns runs fn in the specified network namespace, or in the caller's
// network namespace if netns is nil, returning fn's result.
func inNetns[T any](netns relations.Relation, fn func() T) (T, error) {
	if netns == nil {
		return fn(), nil
	}
	return ops.Execute(fn, netns)
}

// redial returns a newly dialed DNS client connection.
func (p *DnsPool) redial() (*dns.Conn, error) {
	type dialResult struct {
		conn *dns.Conn
		err  error
	}
	res, err := inNetns(p.netns, func() dialResult {
		conn, err := p.dnsclnt.Dial(p.addr)
		return dialResult{conn: conn, err: err}
	})
	if err != nil {
		return nil, err
	}
	return res.conn, res.err
}

// reconnect replaces the network connection of conn with a newly dialed one,
// leaving conn untouched if dialing fails.
func (p *DnsPool) reconnect(conn *dns.Conn) error {
	newconn, err := p.redial()
	if err != nil {
		return err
	}
	_ = conn.Close()
	conn.Conn = newconn.Conn
	return nil
}

// exchange sends msg on conn and returns the reply. As the DNS server might
// have closed an idle connection, exchange reconnects and retries once on
// failure. It never leaves a failed connection behind, as a late reply might
// still arrive on it.
func (p *DnsPool) exchange(ctx context.Context, msg *dns.Msg, conn *dns.Conn) (*dns.Msg, error) {
	r, _, err := p.dnsclnt.ExchangeWithConnContext(ctx, msg, conn)
	if err == nil {
		return r, nil
	}
	if ctx.Err() == nil && p.reconnect(conn) == nil {
		if r, _, err = p.dnsclnt.ExchangeWithConnContext(ctx, msg, conn); err == nil {
			return r, nil
		}
	}
	_ = p.reconnect(conn)
	return nil, err
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

		// Docker's embedded DNS resolves container and service names as single
		// labels, so there is no need for a search list.
		name := dns.Fqdn(name)
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
			msg.SetQuestion(name, addrType)
			var r *dns.Msg
			r, err = p.exchange(ctx, &msg, conn)
			if err != nil {
				return
			}
			if r.Rcode != dns.RcodeSuccess && r.Rcode != dns.RcodeNameError {
				err = fmt.Errorf("ResolveName: query for %q failed with %s",
					name, dns.RcodeToString[r.Rcode])
				return
			}
			for _, rr := range r.Answer {
				switch addrRR := rr.(type) {
				case *dns.A:
					nadanothing = false
					addrs = append(addrs, addrRR.A.String())
				case *dns.AAAA:
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
	task(conn)
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
		_ = conn.Close()
	}
	p.free = nil
}
