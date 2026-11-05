// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package verifier

import (
	"net"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

func TestVerifier(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mobydig/verifier package")
}

var _ = BeforeSuite(func() {
	// Initialize Go's netpoller so its fds don't show up as leaks later.
	conn := Successful(net.ListenPacket("udp", "127.0.0.1:0"))
	Expect(conn.Close()).To(Succeed())
})
