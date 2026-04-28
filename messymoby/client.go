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

// NewClient returns a new Docker client connected to the default socket API
// location on the local host.
func NewClient() *client.Client {
	gi.GinkgoHelper()

	return s.Successful(client.New(
		client.WithHost("unix:///var/run/docker.sock"),
	))
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

// Cleanup removes dead test containers as well as duplicate networks.
func Cleanup(ctx context.Context) {
	cln := NewClient()
	defer func() { _ = cln.Close() }()
	_ = RemoveDeadTestContainers(ctx, cln, MessyMobyLabel)
	_ = RemoveDuplicateTestNetworks(ctx, cln, MessyMobyLabel)
}
