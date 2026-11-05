// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"strings"

	"github.com/siemens/mobydig/dig"
	"github.com/siemens/mobydig/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("rendering", func() {

	It("sorts IPv4 before IPv6 addresses", func() {
		addrs := []types.QualifiedAddressValue{
			{Address: "fe80::1"},
			{Address: "::1"},
			{Address: "10.0.0.2"},
			{Address: "10.0.0.1"},
		}
		sortQualifiedAddresses(addrs)
		Expect(addrs).To(HaveExactElements(
			HaveField("Address", "10.0.0.1"),
			HaveField("Address", "10.0.0.2"),
			HaveField("Address", "::1"),
			HaveField("Address", "fe80::1"),
		))
	})

	It("lists all attached networks without unqualified names", func() {
		var buf bytes.Buffer
		r := &renderer{w: &buf, centerName: "c0", spinner: newSpinner()}
		r.Render([]dig.NamedAddressSet{
			{FQDN: "foo.net_A."},
			{FQDN: "bar.net_B."},
		})
		header, _, _ := strings.Cut(buf.String(), "\n")
		Expect(header).To(SatisfyAll(
			ContainSubstring("net_A"),
			ContainSubstring("net_B"),
		))
	})

})
