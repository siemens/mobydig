// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package types

// NamedAddress represents an FQDN or name, together with an IP address and the
// quality (verification status, [Quality] type) of the address.
type NamedAddress interface {
	QualifiedAddress
	Name() string          // FQDN or name
	NA() NamedAddressValue // returns a copy
}

// QualifiedAddress gives access to qualified address information and also
// allows updating the quality information aspect of an address.
type QualifiedAddress interface {
	Addr() string                                         // returns address
	Qual() Quality                                        // returns Quality
	Err() error                                           // if Quality is Invalid, optional additional error information.
	QA() QualifiedAddressValue                            // returns (a copy of) the qualified address information
	WithNewQuality(q Quality, err error) QualifiedAddress // returns a new and updated qualified address
}

// NamedAddressValue implements a concrete representation of a [NamedAddress].
type NamedAddressValue struct {
	FQDN                  string `json:"fqdn"` // the DNS "name"
	QualifiedAddressValue        // a single associated (resolved) IP network address
}

var _ NamedAddress = (*NamedAddressValue)(nil)

// Name returns the FQDN. Thank you, Go, for nothing.
func (na *NamedAddressValue) Name() string {
	return na.FQDN
}

// NA returns (a copy of) the named address information.
func (na *NamedAddressValue) NA() NamedAddressValue {
	return *na
}

// WithNewQuality returns newly qualified (named) address information.
func (na *NamedAddressValue) WithNewQuality(q Quality, err error) QualifiedAddress {
	qa := na.QA()
	qa.Quality = q
	qa.err = err
	return &NamedAddressValue{
		FQDN:                  na.FQDN,
		QualifiedAddressValue: qa,
	}
}

// QualifiedAddressValue is a network address with an associated quality, such
// as verified, verifying, verified, and invalid.
type QualifiedAddressValue struct {
	Address string  `json:"address"` // a single network IP (v4/v6) address
	Quality Quality `json:"quality"` // quality (validation) state
	err     error   // optional error details for invalid addresses
}

var _ QualifiedAddress = (*QualifiedAddressValue)(nil)

// Addr returns the address.
func (qa *QualifiedAddressValue) Addr() string { return qa.Address }

// Qual return the quality.
func (qa *QualifiedAddressValue) Qual() Quality { return qa.Quality }

// Err returns an optional error that occurred while trying to validate an
// address.
func (qa *QualifiedAddressValue) Err() error { return qa.err }

// QA returns (a copy of) the qualified address information.
func (qa *QualifiedAddressValue) QA() QualifiedAddressValue {
	return *qa
}

// WithNewQuality returns newly qualified address information.
func (qa *QualifiedAddressValue) WithNewQuality(q Quality, err error) QualifiedAddress {
	return &QualifiedAddressValue{
		Address: qa.Address,
		Quality: q,
		err:     qa.err,
	}
}
