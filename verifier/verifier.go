// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package verifier

import (
	"context"

	"github.com/siemens/mobydig/ping"
	"github.com/siemens/mobydig/types"
)

// Verifier verifies a stream of named addresses, caching verification results
// as to avoiding unnecessary duplicate verification attempts. It uses a Pinger
// for verifying the IP addresses.
type Verifier struct {
	news    chan<- types.NamedAddress
	pinger  *ping.Pinger
	checked <-chan types.QualifiedAddress
}

// New returns a new Verifier that verifies addresses from the perspective of
// the specified network namespace with a maximum number of parallel
// verification workers. If the network namespace reference netnsref is zero,
// then the verification will be carried out in the process' original network
// namespace.
func New(size int, netnsref string) (*Verifier, <-chan types.NamedAddress) {
	news := make(chan types.NamedAddress, size)
	pinger, checked := ping.New(size, ping.InNetworkNamespace(netnsref))
	return &Verifier{
		news:    news,
		pinger:  pinger,
		checked: checked,
	}, news
}

// Verify varifies the incoming stream of named addresses until the input
// channel is closed. It then waits for all enqueued verification tasks to
// complete and then closes the output channel returned by New, and finally
// returns.
//
// In case the specified context is cancelled, then Verify will stop pulling off
// new verification tasks and return as soon as possible, closing the output
// channel.
func (v *Verifier) Verify(ctx context.Context, in <-chan types.NamedAddress) {
	addrcache := NewNamedAddressCache()
	// As soon as new validation results trickle in, update the cache so that
	// the cache can inform the consumer of this Validator of the results.
	done := make(chan struct{}, 1) // fire and forget, and never block.
	go func() {
	slurpTasks:
		for {
			select {
			case qaddr, ok := <-v.checked:
				if !ok {
					break slurpTasks
				}
				addrcache.Update(ctx, qaddr.(types.NamedAddress), v.news)
			case <-ctx.Done():
				break slurpTasks
			}
		}
		close(done)
	}()
	// Process incoming named addresses and initiate validation tasks if an
	// address is seen for the first time. Addresses we've already seen, but for
	// different FQDNs, will be directly served if their quality has already
	// been verified. Otherwise, these FQDNs will be put on hold until the
	// verification result becomes available.
slurpPingerVerdicts:
	for {
		select {
		case addr, ok := <-in:
			if !ok {
				break slurpPingerVerdicts
			}
			if addr.Addr() == "" {
				// Pass on yet undug addresses directly to the news channel and wait
				// for more to come in soon.
				select {
				case v.news <- addr:
				case <-ctx.Done():
					break slurpPingerVerdicts
				}
				continue
			}
			if addrcache.Update(ctx, addr, v.news) {
				// Only schedule a validation task the first time we see this
				// particular address.
				v.pinger.ValidateQA(ctx, addr)
			}
		case <-ctx.Done():
			break slurpPingerVerdicts
		}
	}
	v.pinger.StopWait()
	// wait for all verification results to have come through and passed on
	// before calling it a day. In case the context was cancelled we don't wait
	// for the done signal, but immediately close our "outlet".
	select {
	case <-ctx.Done():
	default:
		<-done
	}
	close(v.news)
}
