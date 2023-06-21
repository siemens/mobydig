// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package verifier

import (
	"context"
	"sync"

	"github.com/siemens/mobydig/types"
)

// NamedAddressCache caches named qualified addresses so that unnecessary duplicate
// address validations can be avoided, yet validation results distributed at
// once to all named addresses pending in verification.
type NamedAddressCache struct {
	mu sync.Mutex
	m  map[string]qualityUpdateConsumers // IP address -> list of pending FQDN consumers
}

// NewNamedAddressCache returns a new NamedAddressCache object.
func NewNamedAddressCache() *NamedAddressCache {
	return &NamedAddressCache{
		m: map[string]qualityUpdateConsumers{},
	}
}

// qualityConsumers is a list of FQDNs that map to the same underlying IP
// address and thus want to learn about any updates in that IP address' quality.
type qualityUpdateConsumers struct {
	q         types.Quality
	err       error    // optional error reason for invalid quality
	consumers []string // waiting FQDNs that want to consume quality updates.
}

// Update checks the specified named address to see if it is a new (unverified)
// address which hasn't yet cached. In this case it returns true to signal a new
// address to the caller, so that the caller, for instance, can start validating
// the new address. Update returns false if the (unverified) address has already
// be seen, and the name for this address is cached. If the address is already
// in the cache and its quality is a final verdict of Verified or Invalid, then
// this update is automatically sent to the news consumer for all FQDNs
// associated with this address.
func (c *NamedAddressCache) Update(ctx context.Context, namaddr types.NamedAddress, news chan<- types.NamedAddress) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	addr := namaddr.Addr()
	qc, ok := c.m[addr]
	if !ok {
		// This is the first time we see this address, so we add it to our cache
		// without any further ado.
		//
		// Note: we assume that a new address always enters in qualities
		// Unverified or Verifying, so there will always be a later quality
		// update to be expected.
		c.m[addr] = qualityUpdateConsumers{
			q:         namaddr.Qual(),
			consumers: []string{namaddr.Name()},
		}
		select {
		case news <- namaddr:
		case <-ctx.Done():
		}
		return true
	}
	// So, this address is already known. Now, if this is NOT a quality update
	// by any of the registered consumers for this address, then we're done.
	// Otherwise, update the quality information.
	knownConsumer := false
	fqdn := namaddr.Name()
	for _, consumer := range qc.consumers {
		if consumer == fqdn {
			knownConsumer = true
		}
	}
	if namaddr.Qual() <= qc.q {
		// send an update with the most recent quality known, as the state
		// specified in the Update is already stale. We only need to inform
		// about this specific FQDN, no other consumers affected.
		if !knownConsumer {
			qc.consumers = append(qc.consumers, fqdn)
			c.m[addr] = qc
			select {
			case news <- namaddr.WithNewQuality(qc.q, qc.err).(types.NamedAddress):
			case <-ctx.Done():
			}
		}
		return false
	}
	// update quality
	qc.q = namaddr.Qual()
	// This address is already known, so now check if it is in validation or
	// not. If in validation, then register the current FQDN as a consumer for a
	// later quality update (if not already registered). If already
	// (in)validated, notify all registered consumers.
	var consumers []string
	switch qc.q {
	case types.Unverified, types.Verifying:
		if !knownConsumer {
			qc.consumers = append(qc.consumers, fqdn)
		}
		consumers = qc.consumers
	default:
		// As we've reached one of the terminal qualities, notify all registered
		// consumers and then clear the registration list: all further Update
		// attempts will always be immediately sent for the particular FQDN, as
		// there won't be any quality changes anymore to be send to waiting
		// consumers.
		consumers, qc.consumers = qc.consumers, nil
	}
	c.m[addr] = qc // update cache with most recent quality and consumers.
	// notify all registered consumers of this quality update.
	templ := namaddr.NA()
	templ.Quality = namaddr.Qual()
	for _, consumer := range consumers {
		templ := templ
		templ.FQDN = consumer
		select {
		case news <- &templ:
		case <-ctx.Done(): // bail out immediately.
			return false
		}
	}
	return false
}
