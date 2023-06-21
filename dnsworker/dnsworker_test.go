// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dnsworker

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/miekg/dns"
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

var _ = Describe("DNS client connection pool", func() {

	BeforeEach(func() {
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			Expect(Tasks()).To(BeUniformlyNamespaced())
		})
	})

	It("runs a goroutine-limited set of DNS tasks", NodeTimeout(30*time.Second), func(ctx context.Context) {
		const poolsize = 3

		dnsclnt := dns.Client{}
		// We're never going to contact this DNS "server", we just need just
		// some address so we can allocate some connections.
		pool := Successful(New(ctx, poolsize, &dnsclnt, "127.0.0.1:53"))

		dnsconns := map[*dns.Conn]int{}
		var mu sync.Mutex
		taskfn := func(conn *dns.Conn) {
			mu.Lock()
			defer mu.Unlock()
			count := dnsconns[conn]
			dnsconns[conn] = count + 1
			time.Sleep(time.Second)
		}

		numtasks := poolsize * 2
		for i := 0; i < numtasks; i++ {
			pool.Submit(taskfn)
		}

		pool.StopWait()

		total := 0
		for _, count := range dnsconns {
			total += count
		}
		Expect(total).To(Equal(numtasks), "number of submitted and executed tasks mismatch")
	})

	It("resolves a name", NodeTimeout(30*time.Second), func(ctx context.Context) {
		dnsclnt := dns.Client{}
		// We're never going to contact this DNS "server", we just need just
		// some address so we can allocate some connections.
		pool := Successful(New(ctx, 1, &dnsclnt, "8.8.8.8:53"))
		ch := make(chan []string)

		pool.ResolveName(ctx,
			"a.root-servers.net",
			func(addrs []string, err error) {
				defer GinkgoRecover()
				Expect(err).NotTo(HaveOccurred())
				ch <- addrs
				close(ch)
			})
		Eventually(ch).Should(Receive(Not(BeEmpty())))
		pool.StopWait()
	})

	It("reports resolution failures", NodeTimeout(30*time.Second), func(ctx context.Context) {
		dnsclnt := dns.Client{Net: "udp"}
		pool := Successful(New(ctx, 1, &dnsclnt, "127.0.0.1:1"))
		ch := make(chan []string)

		pool.ResolveName(ctx,
			"tld.rottennet.",
			func(addrs []string, err error) {
				defer GinkgoRecover()
				Expect(err).To(HaveOccurred())
				close(ch)
			})
		Eventually(ch).Should(BeClosed())
		pool.StopWait()
	})

	It("resolves a name from inside a container", NodeTimeout(30*time.Second), func(specctx context.Context) {
		if os.Getuid() != 0 {
			Skip("needs root")
		}

		const name = "foo.net_A"

		mobyw := Successful(moby.New("", nil))
		defer mobyw.Close()
		ctx, cancel := context.WithCancel(specctx)
		defer cancel()
		cizer := whalefriend.New(ctx, []watcher.Watcher{mobyw})
		defer cizer.Close()
		Eventually(mobyw.Ready()).Should(BeClosed())

		disco := discover.Namespaces(
			discover.WithStandardDiscovery(),
			discover.WithContainerizer(cizer),
		)
		testcntr := disco.Containers.FirstWithName("test-test-1")
		Expect(testcntr).NotTo(BeNil(), "missing test-test-1 container")

		dnsclnt := dns.Client{Net: "udp"}
		pool, err := New(context.Background(),
			1,
			&dnsclnt, "127.0.0.11:53",
			InNetworkNamespace(testcntr.Process.Namespaces[model.NetNS].Ref()[0]))
		Expect(err).NotTo(HaveOccurred())
		defer pool.StopWait()

		ch := make(chan struct{})
		pool.ResolveName(specctx,
			name,
			func(addrs []string, err error) {
				defer GinkgoRecover()
				Expect(err).NotTo(HaveOccurred())
				Expect(addrs).NotTo(BeEmpty())
				close(ch)
			})
		Eventually(ch).Should(BeClosed())
	})

})
