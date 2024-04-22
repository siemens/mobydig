[![Siemens](https://img.shields.io/badge/github-siemens-009999?logo=github)](https://github.com/siemens)
[![Industrial Edge](https://img.shields.io/badge/github-industrial%20edge-e39537?logo=github)](https://github.com/industrial-edge)
[![Edgeshark](https://img.shields.io/badge/github-Edgeshark-003751?logo=github)](https://github.com/siemens/edgeshark)

# Moby Dig

[![PkgGoDev](https://pkg.go.dev/badge/github.com/siemens/mobydig)](https://pkg.go.dev/github.com/siemens/mobydig)
[![GitHub](https://img.shields.io/github/license/siemens/mobydig)](https://img.shields.io/github/license/siemens/mobydig)
![build and test](https://github.com/siemens/mobydig/workflows/build%20and%20test/badge.svg?branch=main)
![file descriptors](https://img.shields.io/badge/go%20routines-not%20leaking-success)
[![Go Report Card](https://goreportcard.com/badge/github.com/siemens/mobydig)](https://goreportcard.com/report/github.com/siemens/mobydig)
![Coverage](https://img.shields.io/badge/Coverage-84.4%25-brightgreen)

`mobydig` (dig as in DNS dig) is a consumable Golang module (including a demo
CLI) for diagnosing the reachability of other containers on Docker custom
networks directly reachable from a particular Docker container:
- on these directly attached custom networks, what are the DNS names of other
  containers and (Docker compose) services?
- are the corresponding IP addresses reachable?

`mobydig` is part of the "Edgeshark" project that consist of several
repositories:
- [Edgeshark Hub repository](https://github.com/siemens/edgeshark)
- [G(h)ostwire discovery service](https://github.com/siemens/ghostwire)
- [Packetflix packet streaming service](https://github.com/siemens/packetflix)
- [Containershark Extcap plugin for
  Wireshark](https://github.com/siemens/cshargextcap)
- support modules:
  - [csharg (CLI)](https://github.com/siemens/csharg)
  - 🖝 **mobydig** 🖜
  - [ieddata](https://github.com/siemens/ieddata)

## Usage

Think of `mobydig` as a (pure Go) combination of `dig` and `ping` in order to
dig up the IP addresses associated with other containers and services and then
to (in)validate these IP addresses. But without the need to install `dig` and
`ping` into your containers. This module is designed for consumption by other
modules and thus the `mobydig` command rather is a show case.

In the following example, the container named `test_test_1` is taken as the
starting point. `mobydig` first determines the custom networks `test_test_1` is
connected to, then all other containers also attached to at least one of these
custom networks. Next, the DNS service and container names are dug to which
Docker's embedded DNS resolver will respond, and then finally all resulting IP
addresses pinged.

```bash
$ # set up a testbed with some custom networks and a couple of containers
$ ./test/up # tear down using ./test/down
$ go run -exec sudo ./cmd/mobydig/ test-test-1
```
```text
networks attached to container test-test-1: net_A net_B net_C
DNS names for containers/services on any attached network
   26105065c679        ✔ 172.24.0.4 
   3ab1c736c330        ✔ 172.24.0.2 
   974fd4a0c59f        ✔ 172.23.0.2 
   bar                 ✔ 172.23.0.2 
   foo                 ✔ 172.24.0.2   ✔ 172.24.0.4 
   test-bar-1          ✔ 172.23.0.2 
   test-foo-1          ✔ 172.24.0.2 
   test-foo-2          ✔ 172.24.0.4 
DNS names for containers/services on network net_A
   26105065c679.net_A  ✔ 172.24.0.4 
   3ab1c736c330.net_A  ✔ 172.24.0.2 
   foo.net_A           ✔ 172.24.0.2   ✔ 172.24.0.4 
   test-foo-1.net_A    ✔ 172.24.0.2 
   test-foo-2.net_A    ✔ 172.24.0.4 
DNS names for containers/services on network net_B
   974fd4a0c59f.net_B  ✔ 172.23.0.2 
   bar.net_B           ✔ 172.23.0.2 
   test-bar-1.net_B    ✔ 172.23.0.2 
DNS names for containers/services on network net_C
   26105065c679.net_C  ✔ 172.22.0.4 
   3ab1c736c330.net_C  ✔ 172.22.0.3 
   foo.net_C           ✔ 172.22.0.3   ✔ 172.22.0.4 
   test-foo-1.net_C    ✔ 172.22.0.3 
   test-foo-2.net_C    ✔ 172.22.0.4
```

(Note: this example uses the test deployment in `test/`.)

## Installation

```sh
go get github.com/siemens/mobydig@latest
```

Note: `ieddata` supports versions of Go 1 that are noted by the [Go release
policy](https://golang.org/doc/devel/release.html#policy), that is, major
versions _N_ and _N_-1 (where _N_ is the current major version).

## Components

`mobydig` can be either used as a CLI tool, or its components integrated in
other tools and applications:

- `Digger` takes a list of Docker network names with the container and
  service names on each network and then digs up the associated IP addresses and
  validates them by pinging them. `Digger` operates from the perspective of any
  arbitrary network namespace, and especially from the network perspective of a
  particular container.

- `Validator` consumes names and IP addresses, as emitted by a `Digger` and
  then validates them using a `Pinger`. In contrast to directly wire up a
  `Digger` and a `Pinger`, a `Validator` uses a cache in order to avoid
  duplicate validations when multiple DNS names resolve to the same address(es).

- `Pinger` (in)validates IP addresses from the perspective of any arbitrary
  network namespace inside a Linux host, while live streaming its results over a
  Go channel.

- `DnsPool` operates a limited of eager DNS workers who like to resolve FQDNs
  into their associated IP address(es) from the perspective of any arbitrary
  network namespace inside a Linux host and then stream their findings live over
  a Go channel. `DnsPool` workers can also be deployed in queries for other RRs
  than just `A` and `AAAA` RRs.

## Testing

> The tests require "docker composer" v2 and fail for "docker-composer" v1 due
> to the change in container naming from v1→v2. With Docker CE on Ubuntu or
> Debian, install with `sudo apt-get install docker-compose-plugin` if not
> automatically installed already.

While some of `mobydig`s unit tests can successfully run as an ordinary user,
many tests require root rights. Also, a Docker engine is required for the tests
to successfully complete (well, this package is about Docker's integrated DNS
resolver to quite some extend anyway):

```bash
make test
```

## VSCode Tasks

The included `mobydig.code-workspace` defines the following tasks:

- **View Go module documentation** task: installs `pkgsite`, if not done already
  so, then starts `pkgsite` and opens VSCode's integrated ("simple") browser to
  show the go-plugger/v2 documentation.

- **Build workspace** task: builds all, including the shared library test
  plugin.

- **Run all tests with coverage** task: does what it says on the tin and runs
  all tests with coverage.

#### Aux Tasks

- _pksite service_: auxilliary task to run `pkgsite` as a background service
  using `scripts/pkgsite.sh`. The script leverages browser-sync and nodemon to
  hot reload the Go module documentation on changes; many thanks to @mdaverde's
  [_Build your Golang package docs
  locally_](https://mdaverde.com/posts/golang-local-docs) for paving the way.
  `scripts/pkgsite.sh` adds automatic installation of `pkgsite`, as well as the
  `browser-sync` and `nodemon` npm packages for the local user.
- _view pkgsite_: auxilliary task to open the VSCode-integrated "simple" browser
  and pass it the local URL to open in order to show the module documentation
  rendered by `pkgsite`. This requires a detour via a task input with ID
  "_pkgsite_".

## Make Targets

- `make`: lists all targets.
- `make coverage`: runs all tests with coverage and then **updates the coverage
  badge in `README.md`**.
- `make pkgsite`: installs [`x/pkgsite`](golang.org/x/pkgsite/cmd/pkgsite), as
  well as the [`browser-sync`](https://www.npmjs.com/package/browser-sync) and
  [`nodemon`](https://www.npmjs.com/package/nodemon) npm packages first, if not
  already done so. Then runs the `pkgsite` and hot reloads it whenever the
  documentation changes.
- `make report`: installs
  [`@gojp/goreportcard`](https://github.com/gojp/goreportcard) if not yet done
  so and then runs it on the code base.
- `make test`: runs all tests.

> ⚠ Make sure to **disable parallel test execution** using `-p 1` when testing
> multiple packages, as in the case for `./...`. It is not possible to run
> multiple package tests simultaneously as the multiple docker-compose instances
> will trip on each other happily and create multiple networks with the same
> name, as well as mix and match containers by name and then completely mess up.

## Ouchies ("Lessons Learnt")

In the tradition of [CuriousMarc](youtube.com/curiousmarc)'s "ouchies":

- Forgetting or not knowing that `go test ./...` runs all tests in parallel, so
  that the same docker compose project harness gets created multiple times in
  parallel. See next item why this is considered to be harmful.

- _Names_ of Docker networks are not unique, as opposed to Docker container
  names alway being unique: it is possible to create multiple _different_ Docker
  networks with the _same_ name, yet _unique IDs_. See also `@moby/moby` issue
  [The docker daemon API lets you create two networks with the same name
  #18864](https://github.com/moby/moby/issues/18864).  

## Design Patterns

- Rob Pike's [Self-referential functions and the design of
  options](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html).

- Dave Cheney's [Functional options for friendly
  APIs](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis).

# Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## License and Copyright

(c) Siemens AG 2023

[SPDX-License-Identifier: MIT](LICENSE)
