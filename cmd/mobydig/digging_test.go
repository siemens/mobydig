// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/thediveo/safe"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/namspill"
)

var _ = Describe("digging and reporting", func() {

	BeforeEach(func() {
		goodgos := Goroutines()
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Goroutines).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			Eventually(Filedescriptors).WithTimeout(3 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds))
			Expect(Tasks()).To(BeUniformlyNamespaced())
		})
	})

	It("reports an unknown container without usage", NodeTimeout(30*time.Second), func(ctx context.Context) {
		var out, errout safe.Buffer
		cmd := newRootCmd()
		cmd.SetArgs([]string{"test-nonexisting-1"})
		cmd.SetOut(&out)
		cmd.SetErr(&errout)
		Expect(cmd.ExecuteContext(ctx)).To(MatchError(ContainSubstring("cannot discover attached networks")))
		Expect(out.String()).To(ContainSubstring("inspecting container test-nonexisting-1"))
		Expect(errout.String()).To(SatisfyAll(
			ContainSubstring("Error: cannot discover"),
			Not(ContainSubstring("Usage:"))))
	})

	It("logs debug output to stderr", NodeTimeout(30*time.Second), func(ctx context.Context) {
		DeferCleanup(slog.SetDefault, slog.Default())
		var out, errout safe.Buffer
		cmd := newRootCmd()
		cmd.SetArgs([]string{"--debug", "test-nonexisting-1"})
		cmd.SetOut(&out)
		cmd.SetErr(&errout)
		Expect(cmd.ExecuteContext(ctx)).To(HaveOccurred())
		Expect(errout.String()).To(ContainSubstring("debug logging enabled"))
		Expect(out.String()).NotTo(ContainSubstring("debug logging enabled"))
	})

	It("digs and verifies the names around a container", NodeTimeout(60*time.Second), func(ctx context.Context) {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
		var out safe.Buffer
		cmd := newRootCmd()
		cmd.SetArgs([]string{"test-test-1"})
		cmd.SetOut(&out)
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
