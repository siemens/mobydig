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

// RemoveDeadTestContainers removes exited or created containers carrying the
// specified label. An empty label name removes nothing. Containers that
// vanished in the meantime aren't reported as errors.
func RemoveDeadTestContainers(ctx context.Context, cln *client.Client, labelname string) error {
	exited, err := cln.ContainerList(ctx, client.ContainerListOptions{
		Filters: make(client.Filters).Add("status", "exited"),
	})
	if err != nil {
		return err
	}
	created, err := cln.ContainerList(ctx, client.ContainerListOptions{
		Filters: make(client.Filters).Add("status", "created"),
	})
	if err != nil {
		return err
	}
	deads := append(exited.Items, created.Items...)
	var errs []error
	for _, dead := range deads {
		if _, ok := dead.Labels[labelname]; !ok || labelname == "" {
			continue
		}
		if _, err := cln.ContainerRemove(ctx, dead.ID, client.ContainerRemoveOptions{Force: true}); err != nil && !cerrdefs.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
