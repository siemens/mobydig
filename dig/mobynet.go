// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package dig

// DockerNetwork describes a single Docker network in terms of its name, as well
// as the DNS labels of the attached containers and associated service names.
type DockerNetwork struct {
	Label  string   `json:"label"`  // name of Docker network used as DNS "TLD" label.
	Labels []string `json:"labels"` // container and service/alias names used as DNS labels.
}

// AllFQDNsOnAttachedNetworks returns the list of FQDNs that should be
// addressable from a particular container, based on the list of attached
// networks with DNS labels and container names and aliases (also DNS labels).
//
// A typical means to get the list of attached networks with labels and aliases
// might be mobynet.DiscoverAttachedNames.
func AllFQDNsOnAttachedNetworks(nets []DockerNetwork) []string {
	names := []string{}
	flatnames := map[string]struct{}{}
	for _, net := range nets {
		for _, label := range net.Labels {
			names = append(names, label+"."+net.Label)
			flatnames[label] = struct{}{}
		}
	}
	for flatname := range flatnames {
		names = append(names, flatname)
	}
	return names
}
