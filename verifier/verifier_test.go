// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package verifier

import (
	"context"
	"fmt"
	"time"

	"github.com/siemens/mobydig/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("verifier", func() {

	It("verifies a stream of named addresses", NodeTimeout(30*time.Second), func(ctx context.Context) {
		vf, news := New(2, "")
		in := make(chan types.NamedAddress, 2)
		in <- &types.NamedAddressValue{FQDN: "foo."}
		in <- &types.NamedAddressValue{
			FQDN:                  "foo.",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "127.0.0.1"},
		}
		close(in)
		go vf.Verify(ctx, in)

		var final []types.NamedAddress
		for namaddr := range news {
			if !namaddr.Qual().IsPending() {
				final = append(final, namaddr)
			}
		}
		Expect(final).To(ConsistOf(HaveField("FQDN", "foo.")))
	})

	It("closes its news stream when cancelled", NodeTimeout(30*time.Second), func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		vf, news := New(1, "")
		in := make(chan types.NamedAddress, 3)
		for i := range 3 {
			in <- &types.NamedAddressValue{
				FQDN:                  fmt.Sprintf("foo%d.", i),
				QualifiedAddressValue: types.QualifiedAddressValue{Address: fmt.Sprintf("127.0.0.%d", i+1)},
			}
		}
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			vf.Verify(ctx, in)
			close(done)
		}()
		cancel()
		Eventually(done).Should(BeClosed())
		Eventually(func() bool {
			for {
				select {
				case _, ok := <-news:
					if !ok {
						return true
					}
				default:
					return false
				}
			}
		}).Should(BeTrue())
	})

})
