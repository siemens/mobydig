// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/gosuri/uilive"
	"github.com/thediveo/fdooze/filedesc"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/namspill"
)

var _ = Describe("digging and reporting", func() {

	var out *bytes.Buffer

	BeforeEach(func() {
		goodgos := Goroutines()
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Goroutines).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			// uilive opens /dev/tty on each flush, leaving it to the GC to close.
			Eventually(Filedescriptors).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds, WithTransform(func(fd FileDescriptor) string {
					if pathfd, ok := fd.(*filedesc.PathFd); ok {
						return pathfd.Path()
					}
					return ""
				}, Equal("/dev/tty"))))
			Expect(Tasks()).To(BeUniformlyNamespaced())
		})

		out = &bytes.Buffer{}
		DeferCleanup(func(orig io.Writer) { uilive.Out = orig }, uilive.Out)
		uilive.Out = out
	})

	It("reports an unknown container", NodeTimeout(30*time.Second), func(ctx context.Context) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"test-nonexisting-1"})
		cmd.SetOut(GinkgoWriter)
		cmd.SetErr(GinkgoWriter)
		Expect(cmd.ExecuteContext(ctx)).To(MatchError(ContainSubstring("cannot discover attached networks")))
		Expect(out.String()).To(ContainSubstring("inspecting container test-nonexisting-1"))
	})

	It("digs and verifies the names around a container", NodeTimeout(60*time.Second), func(ctx context.Context) {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
		cmd := newRootCmd()
		cmd.SetArgs([]string{"test-test-1"})
		Expect(cmd.ExecuteContext(ctx)).To(Succeed())
		Expect(out.String()).To(SatisfyAll(
			ContainSubstring("networks attached to container test-test-1"),
			ContainSubstring("net_A"),
			ContainSubstring("net_B"),
			ContainSubstring("net_C"),
			ContainSubstring("test-bar-1.net_B"),
			ContainSubstring("\u2714"),
			Not(ContainSubstring("\u00d7")),
		))
	})

})
