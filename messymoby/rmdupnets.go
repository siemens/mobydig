// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package messymoby

import (
	"context"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// RemoveDuplicateTestNetworks removes duplicate networks, that is multiple
// networks with the same name, yet different IDs. Optionally, removal can be
// limited to networks with a specific label name only.
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
	for netname, netIDs := range netnames {
		if len(netIDs) == 1 {
			continue
		}
		switch netname {
		case "bridge", "host", "none":
			continue
		}
		for _, netID := range netIDs {
			if _, ok := netrscs[netID].Labels[labelname]; ok && labelname != "" {
				_, _ = cln.NetworkRemove(ctx, netID, client.NetworkRemoveOptions{})
			}
		}
	}

	return nil
}
