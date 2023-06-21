// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ping

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/siemens/mobydig/dnsworker"
	"github.com/siemens/mobydig/types"

	"github.com/miekg/dns"
	"github.com/thediveo/lxkns/containerizer/whalefriend"
	"github.com/thediveo/lxkns/discover"
	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/lxkns/ops"
	"github.com/thediveo/lxkns/species"
	"github.com/thediveo/whalewatcher/watcher"
	"github.com/thediveo/whalewatcher/watcher/moby"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/namspill"
	. "github.com/thediveo/success"
)

var _ = Describe("pinger", Ordered, func() {

	var testcntr *model.Container

	BeforeAll(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}

		By("running a container discovery")
		mobyw := Successful(moby.New("", nil))
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(func() { cancel() })
		cizer := whalefriend.New(ctx, []watcher.Watcher{mobyw})
		Eventually(mobyw.Ready()).WithTimeout(2 * time.Second).Should(BeClosed())

		By("waiting for test container to come up")
		Eventually(func() *model.Container {
			disco := discover.Namespaces(
				discover.WithStandardDiscovery(),
				discover.WithContainerizer(cizer),
			)
			testcntr = disco.Containers.FirstWithName("test-test-1") // docker compose v2
			return testcntr
		}).WithTimeout(10*time.Second).ShouldNot(BeNil(), "missing test_/-test_/-1 container")
	})

	BeforeEach(func() {
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			Expect(Tasks()).To(BeUniformlyNamespaced())
		})
	})

	It("handles multiple stops", func() {
		pinger, _ := New(1)
		for i := 0; i < 2; i++ {
			By(fmt.Sprintf("%d round", i+1))
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				pinger.StopWait()
				close(done)
			}()
			Eventually(done).WithTimeout(1 * time.Second).Should(BeClosed())
		}
	})

	It("verifies a named address", NodeTimeout(30*time.Second), func(ctx context.Context) {
		pinger, courtTV := New(1)
		pinger.ValidateQA(ctx, &types.NamedAddressValue{
			FQDN:                  "foobar",
			QualifiedAddressValue: types.QualifiedAddressValue{Address: "localhost"},
		})
		Eventually(courtTV).WithTimeout(5 * time.Second).Should(Receive(
			HaveValue(Equal(types.NamedAddressValue{
				FQDN: "foobar",
				QualifiedAddressValue: types.QualifiedAddressValue{
					Address: "localhost",
					Quality: types.Verified,
				},
			}))))
		pinger.StopWait()
		Eventually(courtTV).Should(BeClosed())
	})

	It("cancels address culture", NodeTimeout(30*time.Second), func(ctx context.Context) {
		netnspath := testcntr.Process.Namespaces[model.NetNS].Ref()[0]
		pinger, courtTV := new(1, 0,
			InNetworkNamespace(netnspath),
			WithCount(1),
			WithInterval(500*time.Millisecond),
			WithThresholdPercentage(1))
		defer pinger.StopWait()
		// Set the context to get cancelled way before the pinger will announce
		// its final verdict.
		ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		go func() {
			// spin off the validation kick-off as it tries to write its
			// intermediate verdict into the non-buffered channel ... where
			// no-one would be listening yet if it were not for starting the
			// validation in a separate goroutine.
			pinger.Validate(ctx, "127.0.0.1")
		}()
		// We should see an intermediate verdict about things being in flight.
		Eventually(courtTV).WithTimeout(1 * time.Second).Should(
			Receive(HaveValue(HaveField("Quality", types.Verifying))))
		cancel()
		// Swallow a "racy" verdict that we cannot avoid.
		wecker := time.NewTimer(time.Second)
		select {
		case <-wecker.C:
		case v := <-courtTV:
			if !wecker.Stop() {
				<-wecker.C
			}
			Expect(v).To(HaveField("Quality", types.Invalid))
		}
		// OH, the semantics of Consistently() in combination with Receive()...
		Consistently(courtTV).WithTimeout(5 * time.Second).ShouldNot(
			Receive())
	})

	It("verifies a stream of addresses", func() {
		pinger, courtTV := New(3)
		inch := make(chan types.QualifiedAddress)
		go func() {
			for i := 0; i < 5; i++ {
				inch <- &types.NamedAddressValue{
					FQDN:                  strconv.Itoa(i),
					QualifiedAddressValue: types.QualifiedAddressValue{Address: "localhost"},
				}
			}
			close(inch)
		}()
		go func() {
			pinger.ValidateStream(inch)
			pinger.StopWait()
		}()
		i := map[string]struct{}{}
		for qa := range courtTV {
			na := qa.(types.NamedAddress).NA()
			if !na.Quality.IsPending() {
				i[na.FQDN] = struct{}{}
			}
		}
		Expect(i).To(HaveLen(5))
	})

	DescribeTable("verifies another container's address",
		func(ctx context.Context, name string, verdict types.Quality) {
			netnspath := testcntr.Process.Namespaces[model.NetNS].Ref()[0]
			pinger, courtTV := New(1,
				InNetworkNamespace(netnspath),
				WithCount(5),
				WithInterval(time.Second),
				WithThresholdPercentage(75))

			testnetnsref := ops.NewTypedNamespacePath(netnspath, species.CLONE_NEWNET)
			var addr string
			// We don't rely on the DnsPool workers to correctly switch network
			// namespaces, so we do it ourselves here... ;)
			Expect(ops.Execute(func() interface{} {
				defer GinkgoRecover()
				By("resolving a container's name from inside another container")

				dnsclnt := dns.Client{}
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				dnspool := Successful(dnsworker.New(ctx, 1, &dnsclnt, "127.0.0.11:53"))
				defer dnspool.StopWait()

				ch := make(chan []string)
				dnspool.ResolveName(ctx, name, func(addrs []string, err error) {
					defer GinkgoRecover()
					By(fmt.Sprintf("resolved container's name %s into address(es) %v", name, addrs))
					ch <- addrs
					close(ch)
				})

				addrs := <-ch
				if len(addrs) == 0 {
					addr = name // expected case; use the non-existing DNS name instead of an IP literal
					return nil
				}
				addr = addrs[0]
				return nil
			}, testnetnsref)).To(Succeed())
			By(fmt.Sprintf("pinging %v", addr))
			pinger.Validate(ctx, addr)
			By("waiting for intermediate verification in-progress verdict")
			Eventually(courtTV).WithTimeout(10 * time.Second).Should(Receive(
				HaveValue(Equal(types.QualifiedAddressValue{
					Address: addr,
					Quality: types.Verifying,
				}))))
			By("waiting for final invalidation verdict")
			Eventually(courtTV).WithTimeout(10*time.Second).Should(Receive(
				HaveValue(Equal(types.QualifiedAddressValue{
					Address: addr,
					Quality: verdict,
				}))), "waiting for the train that never comes: address should be %s", verdict)
			pinger.StopWait()
			Eventually(courtTV).Should(BeClosed())
		},
		Entry("when name is valid", NodeTimeout(30*time.Second), "test-foo-1.net_A", types.Verified),
		Entry("when name is invalid", NodeTimeout(30*time.Second), "test-fool-1.net_A", types.Invalid),
	)

})
