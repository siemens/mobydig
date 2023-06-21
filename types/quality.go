// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package types

import "fmt"

// Quality indicates the "quality" of a network address, such as unverified,
// verified, et cetera.
type Quality int

// The validation qualities of a network address.
const (
	Unverified Quality = iota // address neither in verification nor verified.
	Verifying                 // address in verification.
	Invalid                   // address could not be successfully verified.
	Verified                  // address successfully verified.
)

// String returns the clear-text representation of a Quality value.
func (q Quality) String() string {
	switch q {
	case Unverified:
		return "unverified"
	case Verifying:
		return "verifying"
	case Verified:
		return "verified"
	case Invalid:
		return "invalid"
	}
	return fmt.Sprintf("Quality(%d)", q)
}

// IsPending returns true as long as an address hasn't been either successfully
// or unsuccessfully verified.
func (q Quality) IsPending() bool {
	switch q {
	case Unverified, Verified:
		return true
	default:
		return false
	}
}
