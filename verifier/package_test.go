// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package verifier

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVerifier(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mobydig/verifier package")
}
