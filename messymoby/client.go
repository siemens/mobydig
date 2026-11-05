// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package messymoby

import (
	"context"
	"os/exec"

	"github.com/moby/moby/client"
	s "github.com/thediveo/success"

	gi "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	gx "github.com/onsi/gomega/gexec"
)

// MessyMobyLabel is the name of a “magic” label for tagging testing-related
// container or network elements.
const MessyMobyLabel = "messymoby"

// NewClient returns a new Docker client connected to the Docker daemon
// specified by the environment, defaulting to the local host's socket API.
func NewClient() *client.Client {
	gi.GinkgoHelper()

	return s.Successful(client.New(client.FromEnv))
}

// DockerCompose executes docker-compose with the specified CLI arguments,
// waiting for it to gracefully finish with exit code 0.
func DockerCompose(ctx context.Context, args ...string) {
	gi.GinkgoHelper()

	args = append([]string{"compose"}, args...)
	dc := exec.Command("docker", args...)
	sess := s.Successful(gx.Start(dc, gi.GinkgoWriter, gi.GinkgoWriter))
	g.Eventually(sess).WithContext(ctx).Should(gx.Exit(0))
}

// Cleanup removes dead test containers as well as duplicate networks, failing
// the current spec or setup node on errors other than already vanished
// containers or networks.
func Cleanup(ctx context.Context) {
	gi.GinkgoHelper()

	cln := NewClient()
	defer func() { _ = cln.Close() }()
	g.Expect(RemoveDeadTestContainers(ctx, cln, MessyMobyLabel)).To(g.Succeed(),
		"cannot remove dead test containers")
	g.Expect(RemoveDuplicateTestNetworks(ctx, cln, MessyMobyLabel)).To(g.Succeed(),
		"cannot remove duplicate test networks")
}
