// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package messymoby

import (
	"context"
	"errors"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// RemoveDuplicateTestNetworks removes all networks carrying the specified label
// whose names are used by multiple networks with different IDs. An empty label
// name removes nothing. Networks that vanished in the meantime aren't reported
// as errors.
func RemoveDuplicateTestNetworks(ctx context.Context, cln *client.Client, labelname string) error {
	nets, err := cln.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return err
	}
	// Build map from network names to network IDs, where the same name might
	// map to multiple IDs.
	netnames := map[string][]string{}        // name -> []ID
	netrscs := map[string]*network.Network{} // ID -> details
	for idx, net := range nets.Items {
		netnames[net.Name] = append(netnames[net.Name], net.ID)
		netrscs[net.ID] = &nets.Items[idx].Network
	}
	// Now remove all networks with duplicate IDs.
	var errs []error
	for netname, netIDs := range netnames {
		if len(netIDs) == 1 {
			continue
		}
		switch netname {
		case "bridge", "host", "none":
			continue
		}
		for _, netID := range netIDs {
			if _, ok := netrscs[netID].Labels[labelname]; !ok || labelname == "" {
				continue
			}
			if _, err := cln.NetworkRemove(ctx, netID, client.NetworkRemoveOptions{}); err != nil && !cerrdefs.IsNotFound(err) {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
