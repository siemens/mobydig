// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/siemens/mobydig/messymoby"
	"github.com/siemens/mobydig/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(NodeTimeout(120*time.Second), func(ctx context.Context) {
	By("Cleaning up test containers and networks")
	messymoby.Cleanup(ctx)
	By("Tearing down any left-over test harness")
	messymoby.DockerCompose(ctx, test.DcTestDnArgs...)
	By("Bringing up the test harness")
	messymoby.DockerCompose(ctx, test.DcTestUpArgs...)

	DeferCleanup(NodeTimeout(60*time.Second), func(ctx context.Context) {
		By("Tearing down our test harness")
		messymoby.DockerCompose(ctx, test.DcTestDnArgs...)
		By("Cleaning up test containers and networks")
		messymoby.Cleanup(ctx)
	})
})

func init() {
	// avoid M0 ending up wedged as it was used during a throw-away namespace
	// switch, but as M0 is special it cannot be killed.
	runtime.LockOSThread()
}

func TestDig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mobydig/dig package")
}
