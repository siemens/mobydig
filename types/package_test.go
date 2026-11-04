// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package types

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mobydig/types package")
}
