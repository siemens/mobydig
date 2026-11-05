// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

import (
	"github.com/siemens/mobydig/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

})
