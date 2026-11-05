// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package types

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("qualified addresses", func() {

	boom := errors.New("boom")

	It("requalifies a qualified address with an error", func() {
		qa := &QualifiedAddressValue{Address: "10.0.0.1", Quality: Verifying}
		Expect(qa.WithNewQuality(Invalid, boom)).To(SatisfyAll(
			HaveField("Address", "10.0.0.1"),
			HaveField("Quality", Invalid),
			HaveField("Err()", MatchError(boom)),
		))
		Expect(qa.Err()).To(Succeed())
	})

	It("requalifies a named address with an error", func() {
		na := &NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: QualifiedAddressValue{Address: "10.0.0.1", Quality: Verifying},
		}
		Expect(na.WithNewQuality(Invalid, boom)).To(SatisfyAll(
			HaveField("FQDN", "foo."),
			HaveField("Address", "10.0.0.1"),
			HaveField("Quality", Invalid),
			HaveField("Err()", MatchError(boom)),
		))
		Expect(na.Err()).To(Succeed())
	})

	It("gives access to named address details", func() {
		na := &NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: QualifiedAddressValue{Address: "10.0.0.1", Quality: Verified},
		}
		Expect(na.Name()).To(Equal("foo."))
		Expect(na.Addr()).To(Equal("10.0.0.1"))
		Expect(na.Qual()).To(Equal(Verified))
		Expect(na.QA()).To(Equal(QualifiedAddressValue{Address: "10.0.0.1", Quality: Verified}))
		Expect(na.NA()).To(Equal(*na))
	})

})
