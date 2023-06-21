// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

import (
	"context"
	"sync"

	"github.com/siemens/mobydig/types"
)

// NamedAddressSet is a DNS FQDN together with a list of associated/resolved
// qualified network addresses.
type NamedAddressSet struct {
	FQDN      string                        `json:"fqdn"`      // the DNS "name"
	Addresses []types.QualifiedAddressValue `json:"addresses"` // associated IP network address(es)
}

// NamedAddressesMap maps DNS FQDNs to their corresponding lists of qualified IP
// addresses. A typical use case for a NamedAddressMap is to consume
// name-address information from an event stream (channel) sending updates as
// names are discovered, resolved into the corresponding IP addresses, and
// finally (in)validated.
type NamedAddressesMap struct {
	m  map[string][]types.QualifiedAddressValue
	mu sync.Mutex
}

// Get returns all named addresses from the map.
func (m *NamedAddressesMap) Get() []NamedAddressSet {
	m.mu.Lock()
	defer m.mu.Unlock()
	sets := make([]NamedAddressSet, 0, len(m.m))
	for name, addrs := range m.m {
		sets = append(sets, NamedAddressSet{
			FQDN:      name,
			Addresses: addrs,
		})
	}
	return sets
}

// NewNamedAddressesMap returns a new and properly initialized
// NamedAddressesMap.
func NewNamedAddressesMap() *NamedAddressesMap {
	return &NamedAddressesMap{
		m: map[string][]types.QualifiedAddressValue{},
	}
}

// Update the map with a NamedAddress, augmenting addresses in case they are yet
// unknown. Known addresses are updated in case they have quality changing as
// follows:
//   - from unverified to verifying
//   - from verifying to either verified or invalid
func (m *NamedAddressesMap) Update(namaddr types.NamedAddress) {
	if namaddr == nil {
		return
	}
	fqdn := namaddr.Name()
	if fqdn == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if qualaddr, ok := m.m[fqdn]; ok {
		addr := namaddr.Addr()
		if addr == "" {
			return
		}
		for idx := range qualaddr {
			if qualaddr[idx].Address == addr {
				if namaddr.Qual() > qualaddr[idx].Quality { // slightly simplified "update" rule
					qualaddr[idx].Quality = namaddr.Qual()
				}
				return
			}
		}
		m.m[fqdn] = append(qualaddr, namaddr.QA())
		return
	}
	addr := namaddr.Addr()
	if addr == "" {
		m.m[fqdn] = []types.QualifiedAddressValue{}
	} else {
		m.m[fqdn] = []types.QualifiedAddressValue{namaddr.QA()}
	}
}

// Track NamedAddress updates received from the specified update channel until
// the channel is closed or the context done. Track only returns after
// processing all updates or when the context is done.
func (m *NamedAddressesMap) Track(ctx context.Context, news <-chan types.NamedAddress) error {
	for {
		select {
		case namaddr, ok := <-news:
			if !ok {
				return nil
			}
			m.Update(namaddr)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
