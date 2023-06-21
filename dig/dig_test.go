// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/siemens/mobydig/types"
	"github.com/siemens/mobydig/verifier"

	"github.com/thediveo/lxkns/containerizer/whalefriend"
	"github.com/thediveo/lxkns/discover"
	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/whalewatcher/watcher"
	"github.com/thediveo/whalewatcher/watcher/moby"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/namspill"
	. "github.com/thediveo/success"
)

var _ = Describe("digging in the (address) dirt", Ordered, func() {

	var testcntr *model.Container

	BeforeAll(NodeTimeout(20*time.Second), func(specctx context.Context) {
		if os.Getuid() != 0 {
			Skip("needs root")
		}

		By("running a container discovery")
		mobyw := Successful(moby.New("", nil))
		ctx, cancel := context.WithCancel(specctx)
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
		}).WithContext(specctx).ShouldNot(BeNil(), "missing test-test-1 container")
	})

	BeforeEach(func() {
		goodgos := Goroutines()
		DeferCleanup(func() {
			// cancelling a dig can take some time for all associated goroutines
			// to finally terminate...
			Eventually(Goroutines).Within(3 * time.Second).ProbeEvery(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			// safeguard to catch incorrect OS-level thread locking and
			// unlocking across switching network namespaces.
			Expect(Tasks()).To(BeUniformlyNamespaced())
		})
	})

	It("digs addresses of an FQDN", NodeTimeout(30*time.Second), func(ctx context.Context) {
		netnsref := testcntr.Process.Namespaces[model.NetNS].Ref()[0]
		digger, diggernews := Successful2R(New(4, netnsref))
		Expect(digger).NotTo(BeNil())

		By("running a verifier")
		verifier, news := verifier.New(4, netnsref)
		vfdone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			verifier.Verify(ctx, diggernews)
			close(vfdone)
		}()

		By("digging a network")
		go func() {
			digger.DigNetworks(ctx,
				[]DockerNetwork{
					{
						Label:  "net_A",
						Labels: []string{"foo"},
					},
				})
		}()

		By("waiting for validation")
		Eventually(news).WithContext(ctx).Should(Receive(
			HaveValue(And(
				HaveField("FQDN", Equal("foo.net_A.")),
				HaveField("QualifiedAddressValue.Quality", Equal(types.Verified)),
			))))

		By("winding down")
		digger.StopWait()
		Eventually(news).Within(5 * time.Second).Should(BeClosed())
		Eventually(vfdone).Within(5 * time.Second).Should(BeClosed())
	})

	It("digs and verifies", NodeTimeout(30*time.Second), func(ctx context.Context) {
		nets := []DockerNetwork{
			{
				Label:  "net_A",
				Labels: []string{"foo", "test-foo-1", "test-foo-2"},
			},
			{
				Label:  "net_B",
				Labels: []string{"bar", "test-bar-1"},
			},
			{
				Label:  "net_C",
				Labels: []string{"foo", "test-foo-1", "test-foo-2"},
			},
		}
		netnsref := testcntr.Process.Namespaces[model.NetNS].Ref()[0]
		digger, diggernews := Successful2R(New(4, netnsref))
		Expect(digger).NotTo(BeNil())

		By("running a verifier")
		verifier, news := verifier.New(4, netnsref)
		vfdone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			verifier.Verify(ctx, diggernews)
			close(vfdone)
		}()

		By("digging a network")
		go func() {
			defer GinkgoRecover()
			digger.DigNetworks(ctx, nets)
			digger.StopWait()
		}()

		By("consuming named addresses")
		m := NewNamedAddressesMap()
		Eventually(func() bool {
			namaddr, ok := <-news
			if ok {
				By(fmt.Sprintf("updating FQDN %s, IP %s, Q %s",
					namaddr.Name(), namaddr.Addr(), namaddr.Qual()))
				m.Update(namaddr)
			}
			return ok
		}).WithContext(ctx).Should(BeFalse(), "missing signal that digging has finished")
		Expect(m.m).NotTo(BeEmpty())
		Expect(m.m).To(HaveEach(HaveEach(HaveField("Quality", Equal(types.Verified)))))
		Expect(vfdone).Should(BeClosed())
	})

	It("cancels digging and verifying", NodeTimeout(30*time.Second), func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		nets := []DockerNetwork{
			{
				Label:  "net_A",
				Labels: []string{"foo", "test-foo-1", "test-foo-2"},
			},
		}
		netnsref := testcntr.Process.Namespaces[model.NetNS].Ref()[0]
		digger, diggernews := Successful2R(New(1, netnsref))
		Expect(digger).NotTo(BeNil())

		By("running a verifier")
		verifier, _ := verifier.New(1, netnsref)
		go func() {
			defer GinkgoRecover()
			verifier.Verify(ctx, diggernews)
		}()

		By("digging a network")
		go func() {
			defer GinkgoRecover()
			digger.DigNetworks(ctx, nets)
			digger.StopWait()
		}()

		By("cancelling the context")
		cancel()
		// ...and let the goroutine leak detector do its work!
	})

})
