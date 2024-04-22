// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package messymoby

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// RemoveDeadTestContainers removes dead or stopped containers. Optionally,
// removal can be limited to networks with a specific label name only.
func RemoveDeadTestContainers(ctx context.Context, cln *client.Client, labelname string) error {
	deads, err := cln.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "status", Value: "exited"}),
	})
	if err != nil {
		return err
	}
	created, err := cln.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "status", Value: "created"}),
	})
	if err != nil {
		return err
	}
	deads = append(deads, created...)
	for _, dead := range deads {
		if _, ok := dead.Labels[labelname]; ok && labelname != "" {
			cln.ContainerRemove(ctx, dead.ID, container.RemoveOptions{Force: true})
		}
	}
	return nil
}
