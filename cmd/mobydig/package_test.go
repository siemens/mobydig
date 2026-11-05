// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMobydig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mobydig command")
}
