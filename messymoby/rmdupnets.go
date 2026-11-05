// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package messymoby

import (
	"context"
	"errors"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"
)

// RemoveDuplicateTestNetworks removes all networks carrying the specified label
// whose names are used by multiple of these labelled networks with different
// IDs. An empty label name removes nothing. Networks that vanished in the
// meantime aren't reported as errors.
func RemoveDuplicateTestNetworks(ctx context.Context, cln *client.Client, labelname string) error {
	if labelname == "" {
		return nil
	}
	nets, err := cln.NetworkList(ctx, client.NetworkListOptions{
		Filters: make(client.Filters).Add("label", labelname),
	})
	if err != nil {
		return err
	}
	// Build map from network names to network IDs, where the same name might
	// map to multiple IDs.
	netnames := map[string][]string{} // name -> []ID
	for _, net := range nets.Items {
		netnames[net.Name] = append(netnames[net.Name], net.ID)
	}
	var errs []error
	for _, netIDs := range netnames {
		if len(netIDs) == 1 {
			continue
		}
		for _, netID := range netIDs {
			if _, err := cln.NetworkRemove(ctx, netID, client.NetworkRemoveOptions{}); err != nil && !cerrdefs.IsNotFound(err) {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
