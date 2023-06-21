// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Docker networks", func() {

	It("creates a FQDN bubble", func() {
		cnet := []DockerNetwork{
			{
				Label:  "net_A",
				Labels: []string{"foo", "foo_1", "foo_2"},
			},
			{
				Label:  "net_B",
				Labels: []string{"bar", "bar_1"},
			},
		}
		names := AllFQDNsOnAttachedNetworks(cnet)
		Expect(names).To(ConsistOf(
			"foo", "foo_1", "foo_2",
			"foo.net_A", "foo_1.net_A", "foo_2.net_A",
			"bar", "bar_1", "bar.net_B", "bar_1.net_B",
		))
	})

})
