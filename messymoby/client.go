// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package messymoby

import (
	"context"
	"os/exec"

	"github.com/docker/docker/client"
	"github.com/onsi/gomega/gexec"

	gi "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	s "github.com/thediveo/success"
)

// MessyMobyLabel is the name of a “magic” label for tagging testing-related
// container or network elements.
const MessyMobyLabel = "messymoby"

// NewClient returns a new Docker client connected to the default socket API
// location on the local host.
func NewClient() *client.Client {
	gi.GinkgoHelper()

	return s.Successful(client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.WithAPIVersionNegotiation(),
	))
}

// DockerCompose executes docker-compose with the specified CLI arguments,
// waiting for it to gracefully finish with exit code 0.
func DockerCompose(ctx context.Context, args ...string) {
	gi.GinkgoHelper()

	args = append([]string{"compose"}, args...)
	dc := exec.Command("docker", args...)
	sess := s.Successful(gexec.Start(dc, gi.GinkgoWriter, gi.GinkgoWriter))
	g.Eventually(sess).WithContext(ctx).Should(gexec.Exit(0))
}

// Cleanup removes dead test containers as well as duplicate networks.
func Cleanup(ctx context.Context) {
	cln := NewClient()
	defer cln.Close()
	RemoveDeadTestContainers(ctx, cln, MessyMobyLabel)
	RemoveDuplicateTestNetworks(ctx, cln, MessyMobyLabel)
}
