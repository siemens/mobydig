// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ping

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/siemens/mobydig/types"

	"github.com/gammazero/workerpool"
	"github.com/go-ping/ping"
	"github.com/thediveo/lxkns/ops"
	"github.com/thediveo/lxkns/ops/relations"
	"github.com/thediveo/lxkns/species"
)

// Pinger validates IP addresses by pinging them and then streaming the final
// [types.QualifiedAddress] verdicts to a result/output channel (kind of
// “IT-court TV”). Pingers use a goroutine-limited worker pool.
type Pinger struct {
	count               int           // number of pings to send.
	interval            time.Duration // distance between pings.
	thresholdPercentage uint          // percentage of successful pings for valid IP address.
	unprivileged        bool          // if true, uses UDP-based pings instead of privileged ICMPs.

	netns    relations.Relation          // network namespace to ping from, or nil.
	workers  *workerpool.WorkerPool      // DNS workers for running incoming validation jobs concurrently.
	courtTV  chan types.QualifiedAddress // results/status stream channel.
	stopOnce sync.Once
}

// PingerOption can be passed to New when creating new Pinger objects.
type PingerOption func(*Pinger)

// New returns a new [Pinger] with a maximum worker pool of the specified size
// as well as a “verdict stream”. The verdict channel will not only send the
// final IP address verdicts, but also the initial and yet unverified IP
// addresses as they get submitted for ping court verdicts.
//
// The new pinger defaults to pinging 3 times at intervals of 1s between each
// ping. The validity threshold defaults to 50(%).
//
// The pinger can be configured during creation using several option:
//   - [WithCount]
//   - [WithInterval]
//   - [WithThresholdPercentage]
//   - [AsUnprivileged]
//
// To operate a Pinger in a network namespace different to that of the OS-level
// thread of the caller specify the InNetworkNamespace option and pass it a
// filesystem path that must reference a network namespace (such as
// "/proc/666/ns/net").
func New(size int, options ...PingerOption) (*Pinger, <-chan types.QualifiedAddress) {
	return new(size, size, options...)
}

// new returns a new [Pinger] with a maximum worker pool of the specified size and
// a “verdict stream” with the specified buffer size.
func new(workersize int, chansize int, options ...PingerOption) (*Pinger, <-chan types.QualifiedAddress) {
	courtTV := make(chan types.QualifiedAddress, chansize)
	pinger := &Pinger{
		count:               3,
		interval:            time.Second,
		thresholdPercentage: 50,
		workers:             workerpool.New(workersize),
		courtTV:             courtTV,
	}
	for _, opt := range options {
		opt(pinger)
	}
	return pinger, courtTV
}

// InNetworkNamespace optionally runs a [Pinger] inside the network namespace
// referenced by the specified filesystem path.
func InNetworkNamespace(netnsref string) PingerOption {
	return func(p *Pinger) {
		p.netns = ops.NewTypedNamespacePath(netnsref, species.CLONE_NEWNET)
	}
}

// WithCount sets the number of pings for testing reachability of an IP address.
func WithCount(count uint) PingerOption {
	return func(p *Pinger) {
		p.count = int(count)
	}
}

// WithInterval sets the interval between consecutive pings.
func WithInterval(interval time.Duration) PingerOption {
	return func(p *Pinger) {
		p.interval = interval
	}
}

// Unprivileged tells the Pinger to carry out unprivileged pings using UDP
// instead of ICMP packet.
func AsUnprivileged() PingerOption {
	return func(p *Pinger) {
		p.unprivileged = true
	}
}

// WithThresholdPercentage takes a percentage between 0 and 100 that specifies
// the percentage of successful ping responses required in order to validate the
// pinged IP address.
func WithThresholdPercentage(threshold uint) PingerOption {
	if threshold > 100 {
		panic(fmt.Errorf("Pinger: threshold must be a percentage between 0 <= threshold <= 100, got: %d",
			threshold))
	}
	return func(p *Pinger) {
		p.thresholdPercentage = threshold
	}
}

// ValidateStream reads addresses (with optional attachments) to be validated
// from a channel until the channel is closed. It does not return until the
// channel has been closed, so callers typically might run ValidateStream in a
// separate goroutine.
//
// The input channel transmits [types.QualifiedAddress] objects, but with the
// Quality field initially ignored.
func (p *Pinger) ValidateStream(ch <-chan types.QualifiedAddress) {
	p.ValidateStreamContext(context.Background(), ch)
}

// ValidateStreamContext reads addresses (with optional attachments) to be
// validated from a channel until the channel is closed or the specified context
// gets cancelled. It does not return until the channel has been closed or the
// context cancelled, so callers typically might run ValidateStream in a
// separate goroutine.
//
// If the specified context gets cancelled the pending address verfications
// won't be echoed to the verdict stream at all, and in particular not even as
// invalid. However, spurious verfication verdicts might still appear on the
// verdict stream due to uncontrollable order of verdict sending and context
// cancellation detection.
//
// The input channel transmits [QualifiedAddress] objects, but with the Quality
// field initially ignored.
func (p *Pinger) ValidateStreamContext(ctx context.Context, ch <-chan types.QualifiedAddress) {
	for {
		select {
		case addr, ok := <-ch:
			if !ok {
				return
			}
			qualityAddr := addr.WithNewQuality(types.Verifying, nil)
			p.validate(ctx, qualityAddr)
		case <-ctx.Done():
			return
		}
	}
}

// Validate the specified IP address by pinging it. The verdict is then sent to
// the channel returned together with the newly created [Pinger]. Additionally,
// an initial notice for the address to be validated is also sent beforehand.
//
// If the specified context gets cancelled the pending address verfications
// won't be echoed to the verdict stream at all, and in particular not even as
// invalid. However, spurious verfication verdicts might still appear on the
// verdict stream due to uncontrollable order of verdict sending and context
// cancellation detection.
//
// Please note that you should use IP address literals instead of DNS names in
// case you want precise control over the specific IP address to validate. If
// you instead use DNS names and if the name resolves into multiple IP
// addresses, then you're effectively validating the DNS name, but not a
// particular IP address.
//
// An IP address is considered to be invalid if the percentage of successfully
// received ping replies doesn't reach or cross the Pinger's threshold. This
// allows for some legroom.
//
// The validation process is automatically aborted when the specified context
// either meets its deadline or gets cancelled. The IP address is then
// considered to be Invalid.
func (p *Pinger) Validate(ctx context.Context, addr string) {
	p.validate(ctx, &types.QualifiedAddressValue{
		Address: addr,
		Quality: types.Verifying,
	})
}

// ValidateQA validates the specified [types.QualifiedAddress] and works
// otherwise like [Validate] for a plain address string.
//
// If the specified context gets cancelled the pending address verfications
// won't be echoed to the verdict stream at all, and in particular not even as
// invalid. However, spurious verfication verdicts might still appear on the
// verdict stream due to uncontrollable order of verdict sending and context
// cancellation detection.
//
// The validation process is automatically aborted when the specified context
// either meets its deadline or gets cancelled. The IP address is then
// considered to be Invalid.
func (p *Pinger) ValidateQA(ctx context.Context, addr types.QualifiedAddress) {
	p.validate(ctx, addr.WithNewQuality(types.Verifying, nil))
}

// validate does the real work of pinging a (yet-un-)qualified address. In order
// to avoid an unnecessary [types.QualifiedAddress] clone, the caller is
// expected to pass in a qualified address with its quality already set to
// Verifying.
//
// The validation process is automatically aborted when the specified context
// either meets its deadline or gets cancelled. The IP address is then
// considered to be Invalid.
//
// If the specified context gets cancelled the pending address verfications
// won't be echoed to the verdict stream at all, and in particular not even as
// invalid. However, spurious verfication verdicts might still appear on the
// verdict stream due to uncontrollable order of verdict sending and context
// cancellation detection.
func (p *Pinger) validate(ctx context.Context, verdict types.QualifiedAddress) {
	// Allow cancelling a blocked address verdict send to avoid leaking
	// goroutines. The downside is that since the order in which select checks
	// for ctx.Done() and a blocked verdict channel is random, so we cannot
	// guarantuee that either never a verdict is sent or the verdict gets always
	// sent.
	select {
	case p.courtTV <- verdict: // not yet the final one ;)
	case <-ctx.Done():
		return
	}
	p.workers.Submit(func() {
		verdict := verdict.WithNewQuality(types.Invalid, nil)
		defer func() {
			// Again, allow cancelling a blocked address verdict send to avoid
			// leaking goroutines.
			select {
			case p.courtTV <- verdict: // final one this time.
			case <-ctx.Done():
				return
			}
		}()
		ping := func() interface{} {
			// A quick and non-blocking check to see if the context has been
			// cancelled before we start our work...
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			pinger, err := ping.NewPinger(verdict.Addr())
			if err != nil {
				return err
			}
			pinger.SetPrivileged(!p.unprivileged)
			pinger.Count = p.count
			pinger.Interval = p.interval
			// Always limit waiting for the last ping to get reflected (or not)!
			pinger.Timeout = time.Duration(int64(p.interval) * int64(p.count+2))
			// While the ping will be running, we need to monitor the context in
			// case it becomes "done" by either getting cancelled or reaching
			// its deadline. The done channel here works "the other way round"
			// in the sense that it terminated the concurrent context
			// monitoring.
			done := make(chan struct{})
			defer close(done)
			go func() {
				select {
				case <-ctx.Done():
					pinger.Stop()
				case <-done:
				}
			}()
			// Now start making some noise...
			if err = pinger.Run(); err != nil {
				return err
			}
			// Was the context done?
			if err := ctx.Err(); err != nil {
				return err
			}
			stats := pinger.Statistics()
			if stats.PacketsRecv < pinger.Count*int(p.thresholdPercentage)/100 {
				return errors.New("no replies or too many losses")
			}
			verdict = verdict.WithNewQuality(types.Verified, nil)
			return nil
		}
		// Run the ping in the requested network namespace, if necessary.
		var err error
		if p.netns != nil {
			// lxkns' ops.Execute differentiates between a namespace switching
			// error and the under switched namespaces called function result.
			// We use this function result to return ping errors, so we now need
			// to use the ping-related error (unless there is an Execute-related
			// error) to trigger the final invalid verdict.
			var pingerr interface{}
			pingerr, err = ops.Execute(ping, p.netns)
			if err == nil && pingerr != nil {
				if fnerr, ok := pingerr.(error); ok {
					err = fnerr
				}
			}
		} else {
			if res := ping(); res != nil {
				err = res.(error)
			}
		}
		if err != nil {
			verdict = verdict.WithNewQuality(verdict.QA().Quality, err)
		}
		// falling off the edge of the disc world, triggering the defer'ed and
		// context-controlled verdict send...
	})
}

// StopWait waits for all queued tasks to get processed and then finally closes
// the court TV channel.
func (p *Pinger) StopWait() {
	p.stopOnce.Do(func() {
		p.workers.StopWait()
		close(p.courtTV)
	})
}
