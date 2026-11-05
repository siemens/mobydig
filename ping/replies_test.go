// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ping

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("reply threshold", func() {

	DescribeTable("checks for enough replies",
		func(received, sent int, threshold uint, enough bool) {
			Expect(enoughReplies(received, sent, threshold)).To(Equal(enough))
		},
		Entry("no replies with low threshold", 0, 3, uint(30), false),
		Entry("one reply with low threshold", 1, 3, uint(30), true),
		Entry("one reply at 50%", 1, 3, uint(50), false),
		Entry("two replies at 50%", 2, 3, uint(50), true),
		Entry("all replies at 100%", 3, 3, uint(100), true),
		Entry("missing reply at 100%", 2, 3, uint(100), false),
		Entry("no replies at 0%", 0, 3, uint(0), true),
	)

})
