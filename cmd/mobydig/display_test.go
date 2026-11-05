// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/siemens/mobydig/dig"
	"github.com/siemens/mobydig/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
)

var _ = Describe("rendering", func() {

	BeforeEach(func() {
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
		})
	})

	It("starts and stops its spinner", func() {
		DeferCleanup(func(orig *time.Duration) { spinnerInterval = orig }, spinnerInterval)
		interval := 10 * time.Millisecond
		spinnerInterval = &interval
		r := newRenderer(io.Discard, "c0")
		first := r.spinner.Spinner()
		Eventually(r.spinner.Spinner).ShouldNot(Equal(first))
		r.Stop()
	})

	It("shows a placeholder while there are no names yet", func() {
		var buf bytes.Buffer
		r := &renderer{w: &buf, centerName: "c0", spinner: newSpinner()}
		r.Render(nil)
		Expect(buf.String()).To(ContainSubstring("inspecting container c0"))
	})

	It("renders addresses of all qualities", func() {
		var buf bytes.Buffer
		r := &renderer{w: &buf, centerName: "c0", spinner: newSpinner()}
		r.Render([]dig.NamedAddressSet{
			{
				FQDN: "foo.",
				Addresses: []types.QualifiedAddressValue{
					{Address: "10.0.0.1", Quality: types.Unverified},
					{Address: "10.0.0.2", Quality: types.Verifying},
					{Address: "10.0.0.3", Quality: types.Verified},
					{Address: "10.0.0.4", Quality: types.Invalid},
				},
			},
		})
		Expect(buf.String()).To(SatisfyAll(
			ContainSubstring("on any attached network"),
			ContainSubstring("? 10.0.0.1"),
			ContainSubstring("10.0.0.2"),
			ContainSubstring("✔ 10.0.0.3"),
			ContainSubstring("× 10.0.0.4"),
		))
	})

	It("renders unresolvable names", func() {
		var buf bytes.Buffer
		r := &renderer{w: &buf, centerName: "c0", spinner: newSpinner()}
		r.Render([]dig.NamedAddressSet{{FQDN: "foo.", Error: "no answers"}})
		Expect(buf.String()).To(ContainSubstring("× unresolvable"))
	})

	DescribeTable("groups FQDNs by network",
		func(fqdn string, group string, label string) {
			g, l := groupAndLabel([]string{"net", "my.net"}, fqdn)
			Expect(g).To(Equal(group))
			Expect(l).To(Equal(label))
		},
		Entry(nil, "foo.", "", "foo"),
		Entry(nil, "foo.net.", "net", "foo"),
		Entry(nil, "foo.my.net.", "my.net", "foo"),
		Entry(nil, "foo.bar.net.", "net", "foo.bar"),
		Entry(nil, "my.net.", "net", "my"),
	)

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
		r.SetNetworks([]string{"net_A", "net_B"})
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
