// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("main", func() {

	It("exits with an error code on failure", func() {
		DeferCleanup(func(args []string, exit func(int)) {
			os.Args = args
			osExit = exit
		}, os.Args, osExit)
		os.Args = []string{"mobydig", "--workers=0", "foo"}
		code := 0
		osExit = func(c int) { code = c }
		Expect(capture(&os.Stderr, main)).To(ContainSubstring("--workers"))
		Expect(code).To(Equal(1))
	})

})
