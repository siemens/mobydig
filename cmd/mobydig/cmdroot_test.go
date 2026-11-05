// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mobydig command", func() {

	DescribeTable("rejects invalid arguments",
		func(args []string, errtext string) {
			cmd := newRootCmd()
			cmd.SetArgs(args)
			cmd.SetOut(GinkgoWriter)
			cmd.SetErr(GinkgoWriter)
			Expect(cmd.Execute()).To(MatchError(ContainSubstring(errtext)))
		},
		Entry("missing container name", []string{}, "accepts 1 arg"),
		Entry("too wide indentation", []string{"--indent=81", "foo"}, "--indentation"),
		Entry("no workers", []string{"--workers=0", "foo"}, "--workers"),
		Entry("too many workers", []string{"--workers=11", "foo"}, "--workers"),
		Entry("too fast spinner", []string{"--spinner=1ms", "foo"}, "--spinner"),
	)

})
