// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package test

// DcTestUpArgs specifies docker-compose CLI args for setting up the test
// harness.
var DcTestUpArgs = []string{
	"-f", "../test/docker-compose.yaml",
	"up",
	"-d",
	"--scale", "foo=2",
}

// DcTestDnArgs specifies docker-compose CLI args for tearing down the test
// harness.
var DcTestDnArgs = []string{
	"-f", "../test/docker-compose.yaml",
	"down",
	"-t", "1",
}
