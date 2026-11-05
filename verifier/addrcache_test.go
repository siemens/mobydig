// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package verifier

import (
	"context"
	"errors"

	"github.com/siemens/mobydig/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("named address cache", func() {

	It("passes the invalidation reason on to late named addresses", func(ctx context.Context) {
		boom := errors.New("boom")
		cache := NewNamedAddressCache()
		news := make(chan types.NamedAddress, 10)

		foo := &types.NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "10.0.0.1"},
		}
		Expect(cache.Update(ctx, foo, news)).To(BeTrue())
		Expect(news).To(Receive())

		Expect(cache.Update(ctx,
			AssignableTo[types.NamedAddress](foo.WithNewQuality(types.Invalid, boom)),
			news)).To(BeFalse())
		Expect(news).To(Receive(SatisfyAll(
			HaveField("FQDN", "foo."),
			HaveField("Quality", types.Invalid),
			HaveField("Err()", MatchError(boom)),
		)))

		bar := &types.NamedAddressValue{
			FQDN:                  "bar.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "10.0.0.1"},
		}
		Expect(cache.Update(ctx, bar, news)).To(BeFalse())
		Expect(news).To(Receive(SatisfyAll(
			HaveField("FQDN", "bar."),
			HaveField("Quality", types.Invalid),
			HaveField("Err()", MatchError(boom)),
		)))
	})

})
