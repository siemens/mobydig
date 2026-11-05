// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ping

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
)

var _ = Describe("pinger options", func() {

	BeforeEach(func() {
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
		})
	})

	It("treats a zero count as one", func() {
		p, _ := newPinger(1, 0, WithCount(0))
		DeferCleanup(p.StopWait)
		Expect(p.count).To(Equal(1))
	})

	It("caps the threshold percentage", func() {
		p, _ := newPinger(1, 0, WithThresholdPercentage(200))
		DeferCleanup(p.StopWait)
		Expect(p.thresholdPercentage).To(Equal(uint(100)))
	})

	It("stays in the caller's network namespace for an empty reference", func() {
		p, _ := newPinger(1, 0, InNetworkNamespace(""))
		DeferCleanup(p.StopWait)
		Expect(p.netns).To(BeNil())
	})

})
