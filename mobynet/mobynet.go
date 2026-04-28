// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package mobynet

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/siemens/mobydig/dig"
)

// DiscoverAttachedNames takes on the position of the “origin” or “center”
// container identified by centerID and then inspects the networks attached to
// this container 0. It then queries the containers attached to the attached
// networks for their container names and aliases.
//
// This implementation even works correctly in situations with multiple Docker
// networks having the same name, yet different IDs. Docker networks are
// different from containers in that network names are not necessarily
// unambiguous, while container names always are.
func DiscoverAttachedNames(ctx context.Context, moby *client.Client, centerID string) ([]dig.DockerNetwork, string, error) {
	// Inspect the specified container in order to get information about the
	// networks the container currently is attached to.
	containerDetails, err := moby.ContainerInspect(ctx, centerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, "", err
	}

	if containerDetails.Container.State.Pid == 0 {
		return nil, "", fmt.Errorf("container '%s' is not running", centerID)
	}

	containerDetails.Container.Name = strings.TrimPrefix(containerDetails.Container.Name, "/") // argh, Docker's "/name" legacy!
	netnsref := fmt.Sprintf("/proc/%d/ns/net", containerDetails.Container.State.Pid)

	// In order to avoid repeated inspection of containers that might be
	// connected to multiple networks the container 0 is also attached to, we
	// will cache all inspection results.
	containerDetailsCache := map[string]container.InspectResponse{}
	// Now inspect all attached networks in order to find out which other
	// containers are attached to them, because these are considered to be
	// reachable from container 0.
	mobyNetworks := make([]dig.DockerNetwork, 0, len(containerDetails.Container.NetworkSettings.Networks))
	for attachedNetName, attachedNet := range containerDetails.Container.NetworkSettings.Networks {
		// Inspecting an attached network gives us all the (other) containers
		// directly attached to that attached network (including container 0).
		attNetDetails, err := moby.NetworkInspect(ctx, attachedNet.NetworkID, client.NetworkInspectOptions{})
		if err != nil {
			return nil, "", err
		}
		if len(attNetDetails.Network.Containers) == 0 {
			continue // do not create return empty networks
		}
		// All the names (DNS labels) on this network: since service names might
		// refer to multiple containers, we cannot use a simple slice, but
		// instead need to ensure that each DNS label will appear only once in
		// the final list. And nobody expects ... Captn Map!
		namesOnNetwork := map[string]struct{}{}
		// Now inspect the containers attached to this network attached to
		// container 0. These additional inspections become necessary, as the
		// attached network inspection doesn't reveal the container aliases, but
		// only the container names ... and not even the container IDs.
		for _, attachedContainer := range attNetDetails.Network.Containers {
			// Well, do not add our own container label to the resulting list.
			if attachedContainer.Name == containerDetails.Container.Name {
				continue
			}
			// the link from a network to an attached container is by container
			// name, but not container ID. Anyway, see if we have something in
			// our cache, otherwise get the ugly container details and then
			// cache them.
			attachedContainerDetails, ok := containerDetailsCache[attachedContainer.Name]
			if !ok {
				details, err := moby.ContainerInspect(ctx, attachedContainer.Name, client.ContainerInspectOptions{})
				if err != nil {
					continue
				}
				attachedContainerDetails = details.Container
				containerDetailsCache[attachedContainer.Name] = attachedContainerDetails
			}
			namesOnNetwork[attachedContainer.Name] = struct{}{}
			for _, alias := range attachedContainerDetails.NetworkSettings.Networks[attachedNetName].Aliases {
				namesOnNetwork[alias] = struct{}{}
			}
		}
		// Add the DNS label-related information about this Docker network to
		// the result.
		dnsLabels := make([]string, 0, len(namesOnNetwork))
		for alias := range namesOnNetwork {
			dnsLabels = append(dnsLabels, alias)
		}
		mobyNetworks = append(mobyNetworks, dig.DockerNetwork{
			Label:  attachedNetName,
			Labels: dnsLabels,
		})
	}
	return mobyNetworks, netnsref, nil
}
