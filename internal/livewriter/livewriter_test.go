// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package livewriter

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("live writer", func() {

	It("replaces previously flushed output", func() {
		var buf bytes.Buffer
		w := New(&buf)
		Expect(fmt.Fprint(w, "foo\nbar\n")).To(Equal(8))
		Expect(buf.Len()).To(BeZero())
		Expect(w.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("foo\nbar\n"))

		buf.Reset()
		Expect(fmt.Fprint(w, "baz\n")).Error().NotTo(HaveOccurred())
		Expect(w.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal(eraseLine + strings.Repeat(eraseLineAbove, 2) + "baz\n"))
	})

	It("accounts for wrapped lines", func() {
		var buf bytes.Buffer
		w := New(&buf)
		w.width = func() int { return 3 }
		Expect(fmt.Fprint(w, "abcdefg\n\x1b[31mabc\x1b[0m\n\n")).Error().NotTo(HaveOccurred())
		Expect(w.Flush()).To(Succeed())

		buf.Reset()
		Expect(w.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal(eraseLine + strings.Repeat(eraseLineAbove, 5)))
	})

	It("accounts for wide characters", func() {
		var buf bytes.Buffer
		w := New(&buf)
		w.width = func() int { return 4 }
		Expect(fmt.Fprint(w, "日本語\n")).Error().NotTo(HaveOccurred())
		Expect(w.Flush()).To(Succeed())

		buf.Reset()
		Expect(w.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal(eraseLine + strings.Repeat(eraseLineAbove, 2)))
	})

	It("erases a trailing partial line", func() {
		var buf bytes.Buffer
		w := New(&buf)
		w.width = func() int { return 3 }
		Expect(fmt.Fprint(w, "foo\nbarbaz")).Error().NotTo(HaveOccurred())
		Expect(w.Flush()).To(Succeed())

		buf.Reset()
		Expect(w.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal(eraseLine + strings.Repeat(eraseLineAbove, 2)))
	})

	It("reports write errors", func() {
		Expect(New(failingWriter{}).Flush()).To(HaveOccurred())
	})

	It("doesn't know the width of non-terminals", func() {
		Expect(termWidth(&bytes.Buffer{})).To(BeZero())
		r, w := Successful2R(os.Pipe())
		DeferCleanup(r.Close)
		DeferCleanup(w.Close)
		Expect(termWidth(w)).To(BeZero())
	})

})
