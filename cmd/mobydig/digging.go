// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/siemens/mobydig/dig"
	"github.com/siemens/mobydig/mobynet"
	"github.com/siemens/mobydig/verifier"

	"github.com/docker/docker/client"
	"github.com/gosuri/uilive"
)

// DigAndReport locates a “starting point” container by its name and then looks
// for networks attached to it. Next, container and service names on these
// networks are discovered, and then these (DNS) names dug up from the
// perspective of the center container. Finally, the addresses are verified by
// pinging them for good or bad.
func DigAndReport(ctx context.Context, startpointName string) error {
	cln, err := client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("cannot connect to the Docker daemon: %w", err)
	}

	// Create an empty (concurrency-safe) result map with named-and-qualified
	// addresses and immediately fire off the rendering goroutine. The rendering
	// will only stop after tracking has finished because the result stream
	// channel has been closed. We then render a final update and end rendering,
	// signalling the end of our activities via renderingDone.
	namaddrs := dig.NewNamedAddressesMap()
	trackingDone := make(chan struct{})
	renderingDone := make(chan struct{})

	go func() {
		// Dunno what uilive's background updating mode using Start() is good
		// for? It may trigger anytime with the rendering into the buffer not
		// yet complete, thus making the terminal output very flickery. So we
		// avoid Start() and instead trigger an explicit flush to the terminal
		// after having completed the rendering.
		term := uilive.New()
		renderer := newRenderer(term, startpointName)
		renderer.Indentation = int(*indentation)
		defer func() {
			renderData(term, renderer, namaddrs)
			renderer.Stop()
			close(renderingDone)
		}()
		renderData(term, renderer, namaddrs)
		ticker := time.NewTicker(20 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				renderData(term, renderer, namaddrs)
			case <-trackingDone:
				return
			}
		}
	}()

	attachedNets, netnsref, err := mobynet.DiscoverAttachedNames(ctx, cln, startpointName)
	if err != nil {
		return fmt.Errorf("cannot discover attached networks and their containers: %w", err)
	}

	// Now lets put the required processing elements and their plumbing in
	// place.
	//
	//   - Digger producing IP addresses from a list of FQDNs.
	//   - Verifier consuming the IPs and checking them, producing "verdicts".
	//   - NamedAddressMap consuming these "verdicts".
	//
	// Rendering is done on the information collected by the NamedAddressMap.
	digger, diggernews, err := dig.New(int(*workerNumber), netnsref)
	if err != nil {
		return fmt.Errorf("cannot dig address information: %w", err)
	}
	verifier, news := verifier.New(int(*workerNumber), netnsref)
	go verifier.Verify(ctx, diggernews)
	go func() {
		_ = namaddrs.Track(ctx, news)
		close(trackingDone)
	}()

	// Finally feed the information about attached networks and their names into
	// the Digger, so they can be processed and move through the different
	// stages. Then close the input stream and wait for all the data to pass the
	// stages and finally get rendered a last time.
	go func() {
		digger.DigNetworks(context.Background(), attachedNets)
		digger.StopWait()
	}()
	<-renderingDone

	return nil
}

// renderData get the current named+verified address data and then renders (and
// flushes) it to the terminal.
func renderData(term *uilive.Writer, r *renderer, data *dig.NamedAddressesMap) {
	r.Render(data.Get())
	term.Flush()
}
