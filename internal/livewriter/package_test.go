// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package livewriter

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLivewriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mobydig/internal/livewriter package")
}

// failingWriter fails all writes.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("D'OH!") }
