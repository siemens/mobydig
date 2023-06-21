// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package mobynet

import (
	"context"
	"time"

	"github.com/docker/docker/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/success"
)

var _ = Describe("Docker networks", func() {

	BeforeEach(func() {
		goodgos := Goroutines()
		DeferCleanup(func() {
			// cancelling a dig can take some time for all associated goroutines
			// to finally terminate...
			Eventually(Goroutines).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
		})
	})

	It("discovers attached networks with their containers", NodeTimeout(30*time.Second), func(ctx context.Context) {
		cln := Successful(client.NewClientWithOpts(
			client.WithHost("unix:///var/run/docker.sock"),
			client.WithAPIVersionNegotiation(),
		))
		defer cln.Close()
		dnets, _ := Successful2R(DiscoverAttachedNames(ctx, cln, "test-test-1"))
		Expect(dnets).To(ContainElements(
			And(
				HaveField("Label", "net_A"),
				HaveField("Labels", ContainElements("foo", "test-foo-1", "test-foo-2")),
			),
			And(
				HaveField("Label", "net_B"),
				HaveField("Labels", ContainElements("bar", "test-bar-1")),
			),
			And(
				HaveField("Label", "net_C"),
				HaveField("Labels", ContainElements("foo", "test-foo-1", "test-foo-2")),
			),
		))
	})

})
