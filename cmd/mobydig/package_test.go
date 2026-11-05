// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/siemens/mobydig/messymoby"
	"github.com/siemens/mobydig/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

func TestMobydig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mobydig command")
}

var _ = BeforeSuite(NodeTimeout(120*time.Second), func(ctx context.Context) {
	dnArgs := composeArgs(test.DcTestDnArgs)
	upArgs := composeArgs(test.DcTestUpArgs)

	By("Cleaning up test containers and networks")
	messymoby.Cleanup(ctx)
	By("Tearing down any left-over test harness")
	messymoby.DockerCompose(ctx, dnArgs...)
	By("Bringing up the test harness")
	messymoby.DockerCompose(ctx, upArgs...)

	DeferCleanup(NodeTimeout(60*time.Second), func(ctx context.Context) {
		By("Tearing down our test harness")
		messymoby.DockerCompose(ctx, dnArgs...)
		By("Cleaning up test containers and networks")
		messymoby.Cleanup(ctx)
	})
})

func init() {
	// avoid M0 ending up wedged as it was used during a throw-away namespace
	// switch, but as M0 is special it cannot be killed.
	runtime.LockOSThread()
}

// composeArgs returns the specified docker compose args with the compose file
// path adjusted to this package's location in the hierarchy.
func composeArgs(args []string) []string {
	args = slices.Clone(args)
	if idx := slices.Index(args, "-f"); idx >= 0 && idx+1 < len(args) {
		args[idx+1] = filepath.Join("..", args[idx+1])
	}
	return args
}

// capture returns everything fn writes to the specified standard stream, such
// as os.Stdout or os.Stderr.
func capture(stream **os.File, fn func()) (out string) {
	GinkgoHelper()
	r, w := Successful2R(os.Pipe())
	var buf bytes.Buffer
	copied := make(chan struct{})
	go func() {
		defer close(copied)
		_, _ = io.Copy(&buf, r)
	}()
	orig := *stream
	*stream = w
	defer func() {
		*stream = orig
		_ = w.Close()
		<-copied
		_ = r.Close()
		out = buf.String()
	}()
	fn()
	return
}
