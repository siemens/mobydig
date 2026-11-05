// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

import (
	"context"
	"errors"

	"github.com/siemens/mobydig/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("named addresses map", func() {

	It("returns copies of its addresses", func() {
		m := NewNamedAddressesMap()
		m.Update(&types.NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "10.0.0.1"},
		})
		namaddrs := m.Get()
		Expect(namaddrs).To(HaveLen(1))
		Expect(namaddrs[0].Addresses).To(HaveLen(1))
		namaddrs[0].Addresses[0].Quality = types.Verified
		Expect(m.Get()[0].Addresses[0].Quality).To(Equal(types.Unverified))
	})

	It("records unresolvable names", func() {
		m := NewNamedAddressesMap()
		m.Update(&types.NamedAddressValue{FQDN: "foo."})
		m.Update(AssignableTo[types.NamedAddress](
			(&types.NamedAddressValue{FQDN: "foo."}).WithNewQuality(types.Invalid, errors.New("no answers"))))
		Expect(m.Get()).To(ConsistOf(SatisfyAll(
			HaveField("FQDN", "foo."),
			HaveField("Addresses", BeEmpty()),
			HaveField("Error", "no answers"),
		)))
	})

	It("ignores incomplete and stale updates", func() {
		m := NewNamedAddressesMap()
		m.Update(nil)
		m.Update(&types.NamedAddressValue{})
		m.Update(&types.NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "10.0.0.1", Quality: types.Verified},
		})
		m.Update(&types.NamedAddressValue{FQDN: "foo."})
		m.Update(&types.NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "10.0.0.1", Quality: types.Verifying},
		})
		m.Update(&types.NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "10.0.0.2"},
		})
		Expect(m.Get()).To(ConsistOf(HaveField("Addresses", ConsistOf(
			types.QualifiedAddressValue{Address: "10.0.0.1", Quality: types.Verified},
			types.QualifiedAddressValue{Address: "10.0.0.2", Quality: types.Unverified},
		))))
	})

	It("tracks updates until the news channel gets closed", func(ctx context.Context) {
		m := NewNamedAddressesMap()
		news := make(chan types.NamedAddress, 2)
		news <- &types.NamedAddressValue{FQDN: "foo."}
		news <- &types.NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "10.0.0.1", Quality: types.Verified},
		}
		close(news)
		Expect(m.Track(ctx, news)).To(Succeed())
		Expect(m.Get()).To(ConsistOf(SatisfyAll(
			HaveField("FQDN", "foo."),
			HaveField("Addresses", ConsistOf(HaveField("Quality", types.Verified))),
		)))
	})

	It("stops tracking when the context gets cancelled", func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		Expect(NewNamedAddressesMap().Track(ctx, make(chan types.NamedAddress))).
			To(MatchError(context.Canceled))
	})

})
