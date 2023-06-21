// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import "github.com/muesli/termenv"

var (
	verifyingAddressStyle = termenv.Style{}.Foreground(termenv.ANSIYellow)
	validAddressStyle     = termenv.Style{}.Foreground(termenv.ANSIGreen)
	invalidAddressStyle   = termenv.Style{}.Foreground(termenv.ANSIRed)
)

var networkNameStyle = termenv.Style{}.Bold()
