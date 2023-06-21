// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"

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

// Render the given named+qualified addresses.
func (r *renderer) Render(na []dig.NamedAddressSet) {
	groups := groupNames(na)
	// If we don't have any name+addressing information yet, show a proxy
	// message.
	if len(groups) == 0 {
		fmt.Fprintf(r.w, "inspecting container %s and its networks...\n", r.centerName)
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
	fmt.Fprintf(r.w, "networks attached to container %s: ", r.centerName)
	for idx, group := range groups {
		if idx == 0 {
			continue // skip unnamed group
		}
		if idx >= 2 {
			fmt.Fprint(r.w, " ")
		}
		fmt.Fprint(r.w, networkNameStyle.Styled(groupName(group[0].FQDN)))
	}
	fmt.Fprintln(r.w)
	// Render the network groups...
	for _, group := range groups {
		gn := groupName(group[0].FQDN)
		switch gn {
		case "":
			fmt.Fprint(r.w, "DNS names for containers/services on any attached network\n")
		default:
			fmt.Fprintf(r.w, "DNS names for containers/services on network %s\n", networkNameStyle.Styled(gn))
		}
		for _, na := range group {
			r.renderGroupDetails(maxlen, na)
		}
	}
}

// renderGroupDetails renders a network group's labels and qualified addresses.
func (r *renderer) renderGroupDetails(labelwidth int, na dig.NamedAddressSet) {
	fmt.Fprintf(r.w, "%-*s%-*s", r.Indentation, "", labelwidth, strings.TrimSuffix(na.FQDN, "."))
	for idx, addr := range na.Addresses {
		if idx > 0 {
			fmt.Fprint(r.w, " ")
		}
		switch addr.Quality {
		case types.Unverified:
			fmt.Fprintf(r.w, " ? %s", addr.Address)
		case types.Verifying:
			fmt.Fprint(r.w, verifyingAddressStyle.Styled(" "+r.spinner.Spinner()+addr.Address+" "))
		case types.Verified:
			fmt.Fprint(r.w, validAddressStyle.Styled(" ✔ "+addr.Address+" "))
		case types.Invalid:
			fmt.Fprint(r.w, invalidAddressStyle.Styled(" × "+addr.Address+" "))
		}
	}
	fmt.Fprintln(r.w)
}

// sortQualifiedAddresses sorts a slice of qualified address in place.
// - IPv4 first, IPv6 ... (embarrassed slience) ... second.
// - sorts by address value.
func sortQualifiedAddresses(addrs []types.QualifiedAddressValue) {
	sort.Slice(addrs, func(a, b int) bool {
		ipA := net.ParseIP(addrs[a].Address)
		ipB := net.ParseIP(addrs[b].Address)
		return bytes.Compare(ipA, ipB) < 0
	})
}

// sortFQDNs sorts a slice of named addresses in place according to their
// grouped labels. That is, sorting order is not lexicographically on the FQDNs,
// but instead first according to network labels (if not present, then assumed
// to be ""), and second according to the service/container labels.
func sortFQDNs(addrs []dig.NamedAddressSet) {
	sort.Slice(addrs, func(a, b int) bool {
		gA, lA := groupAndLabel(addrs[a].FQDN)
		gB, lB := groupAndLabel(addrs[b].FQDN)
		return (gA < gB) || ((gA == gB) && (lA < lB))
	})
}

// groupAndLabel returns the group label and the container/service label
// separately, given an FQDN. If the FQDN consists of only a single label, then
// this is taken to refer to a container/service label and the group label is
// assumed to be "".
func groupAndLabel(fqdn string) (group string, label string) {
	f := strings.Split(strings.TrimSuffix(fqdn, "."), ".")
	if len(f) > 1 {
		return f[1], f[0]
	}
	return "", f[0]
}

// groupName returns the group label of an FQDN, or "" if the FQDN consists of a
// single label only.
func groupName(fqdn string) string {
	group, _ := groupAndLabel(fqdn)
	return group
}

// Note: groupNames modifies the passed addrs in place.
func groupNames(addrs []dig.NamedAddressSet) [][]dig.NamedAddressSet {
	sortFQDNs(addrs)
	groups := [][]dig.NamedAddressSet{}
	var recentGroup []dig.NamedAddressSet
	for _, addr := range addrs {
		gn := groupName(addr.FQDN)
		// if this is the first group ever or we have wandered off into a new
		// group, then allocate a new group.
		if recentGroup == nil || gn != groupName(recentGroup[0].FQDN) {
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
		sortFQDNs(group)
	}
	return groups
}
