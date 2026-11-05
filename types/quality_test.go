// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("address quality", func() {

	DescribeTable("pending qualities",
		func(q Quality, pending bool) {
			Expect(q.IsPending()).To(Equal(pending))
		},
		Entry("unverified", Unverified, true),
		Entry("verifying", Verifying, true),
		Entry("invalid", Invalid, false),
		Entry("verified", Verified, false),
	)

	DescribeTable("textual representation",
		func(q Quality, text string) {
			Expect(q.String()).To(Equal(text))
		},
		Entry(nil, Unverified, "unverified"),
		Entry(nil, Verifying, "verifying"),
		Entry(nil, Invalid, "invalid"),
		Entry(nil, Verified, "verified"),
		Entry(nil, Quality(42), "Quality(42)"),
	)

})
