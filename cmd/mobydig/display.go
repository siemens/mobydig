// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"io"
	"net/netip"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/siemens/mobydig/dig"
	"github.com/siemens/mobydig/types"
)

// renderer renders the terminal display, based on named+qualified address
// information passed to its Render method.
type renderer struct {
	Indentation int
	centerName  string
	w           io.Writer
	spinner     *spinner

	mu       sync.Mutex
	networks []string // names of the networks attached to the center container.
}

// newRenderer returns a Render object rendering to the specified io.Writer.
// centerName identifies the container from which the names+address information
// is seen.
func newRenderer(w io.Writer, centerName string) *renderer {
	sp := newSpinner()
	sp.Start(*spinnerInterval)
	return &renderer{
		centerName: centerName,
		w:          w,
		spinner:    sp,
	}
}

// Stop the renderer's background ticker.
func (r *renderer) Stop() {
	r.spinner.Stop()
}

// SetNetworks sets the names of the networks attached to the center container,
// so that FQDNs can be grouped by network.
func (r *renderer) SetNetworks(names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.networks = slices.Clone(names)
}

// Render the given named+qualified addresses.
func (r *renderer) Render(na []dig.NamedAddressSet) {
	r.mu.Lock()
	networks := r.networks
	r.mu.Unlock()
	groups := groupNames(networks, na)
	// If we don't have any name+addressing information yet, show a proxy
	// message.
	if len(groups) == 0 {
		_, _ = fmt.Fprintf(r.w, "inspecting container %s and its networks...\n", r.centerName)
		return
	}
	// For neat display, determine the length of the longest FQDN in the data to
	// display, so that the addresses column doesn't zig-zag around across
	// different groups.
	maxlen := 0
	for _, group := range groups {
		for _, addr := range group {
			if l := len(addr.FQDN); l > maxlen {
				maxlen = l
			}
		}
	}
	// Render list of attached networks...
	_, _ = fmt.Fprintf(r.w, "networks attached to container %s: ", r.centerName)
	sep := ""
	for _, group := range groups {
		gn := groupName(networks, group[0].FQDN)
		if gn == "" {
			continue
		}
		_, _ = fmt.Fprint(r.w, sep, networkNameStyle.Styled(gn))
		sep = " "
	}
	_, _ = fmt.Fprintln(r.w)
	// Render the network groups...
	for _, group := range groups {
		gn := groupName(networks, group[0].FQDN)
		switch gn {
		case "":
			_, _ = fmt.Fprint(r.w, "DNS names for containers/services on any attached network\n")
		default:
			_, _ = fmt.Fprintf(r.w, "DNS names for containers/services on network %s\n", networkNameStyle.Styled(gn))
		}
		for _, na := range group {
			r.renderGroupDetails(maxlen, na)
		}
	}
}

// renderGroupDetails renders a network group's labels and qualified addresses.
func (r *renderer) renderGroupDetails(labelwidth int, na dig.NamedAddressSet) {
	_, _ = fmt.Fprintf(r.w, "%-*s%-*s", r.Indentation, "", labelwidth, strings.TrimSuffix(na.FQDN, "."))
	if len(na.Addresses) == 0 && na.Error != "" {
		_, _ = fmt.Fprint(r.w, invalidAddressStyle.Styled(" × unresolvable "))
	}
	for idx, addr := range na.Addresses {
		if idx > 0 {
			_, _ = fmt.Fprint(r.w, " ")
		}
		switch addr.Quality {
		case types.Unverified:
			_, _ = fmt.Fprintf(r.w, " ? %s", addr.Address)
		case types.Verifying:
			_, _ = fmt.Fprint(r.w, verifyingAddressStyle.Styled(" "+r.spinner.Spinner()+addr.Address+" "))
		case types.Verified:
			_, _ = fmt.Fprint(r.w, validAddressStyle.Styled(" ✔ "+addr.Address+" "))
		case types.Invalid:
			_, _ = fmt.Fprint(r.w, invalidAddressStyle.Styled(" × "+addr.Address+" "))
		}
	}
	_, _ = fmt.Fprintln(r.w)
}

// sortQualifiedAddresses sorts a slice of qualified address in place.
// - IPv4 first, IPv6 ... (embarrassed silence) ... second.
// - sorts by address value.
func sortQualifiedAddresses(addrs []types.QualifiedAddressValue) {
	slices.SortFunc(addrs, func(a, b types.QualifiedAddressValue) int {
		ipA, _ := netip.ParseAddr(a.Address)
		ipB, _ := netip.ParseAddr(b.Address)
		return ipA.Compare(ipB)
	})
}

// sortFQDNs sorts a slice of named addresses in place according to their
// grouped labels. That is, sorting order is not lexicographically on the FQDNs,
// but instead first according to network labels (if not present, then assumed
// to be ""), and second according to the service/container labels.
func sortFQDNs(networks []string, addrs []dig.NamedAddressSet) {
	sort.Slice(addrs, func(a, b int) bool {
		gA, lA := groupAndLabel(networks, addrs[a].FQDN)
		gB, lB := groupAndLabel(networks, addrs[b].FQDN)
		return (gA < gB) || ((gA == gB) && (lA < lB))
	})
}

// groupAndLabel returns the network name and the container/service label
// separately, given an FQDN and the names of the attached networks. As both
// network and container names might contain dots, the FQDN is matched against
// the network names, preferring the longest match. If the FQDN doesn't end in
// any network name, then it is taken to be a container/service label and the
// network name is assumed to be "".
func groupAndLabel(networks []string, fqdn string) (group string, label string) {
	name := strings.TrimSuffix(fqdn, ".")
	label = name
	for _, network := range networks {
		if l, ok := strings.CutSuffix(name, "."+network); ok && l != "" && len(network) > len(group) {
			group, label = network, l
		}
	}
	return group, label
}

// groupName returns the network name of an FQDN, or "" if the FQDN doesn't end
// in any of the specified network names.
func groupName(networks []string, fqdn string) string {
	group, _ := groupAndLabel(networks, fqdn)
	return group
}

// Note: groupNames modifies the passed addrs in place.
func groupNames(networks []string, addrs []dig.NamedAddressSet) [][]dig.NamedAddressSet {
	sortFQDNs(networks, addrs)
	groups := [][]dig.NamedAddressSet{}
	var recentGroup []dig.NamedAddressSet
	for _, addr := range addrs {
		gn := groupName(networks, addr.FQDN)
		// if this is the first group ever or we have wandered off into a new
		// group, then allocate a new group.
		if recentGroup == nil || gn != groupName(networks, recentGroup[0].FQDN) {
			if recentGroup != nil {
				groups = append(groups, recentGroup)
			}
			recentGroup = []dig.NamedAddressSet{}
		}
		sortQualifiedAddresses(addr.Addresses)
		recentGroup = append(recentGroup, addr)
	}
	if recentGroup != nil {
		groups = append(groups, recentGroup)
	}
	for _, group := range groups {
		sortFQDNs(networks, group)
	}
	return groups
}
