// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

// Yet another (braille) spinner.

package main

import (
	"sync"
	"time"
)

// spinner is yet another blindingly simple spinner; just enough to get the job
// done, no bells, no frills.
type spinner struct {
	ticker *time.Ticker
	phases []string
	done   chan struct{}
	mu     sync.Mutex
	phase  int
}

// newSpinner returns a new spinner; later call the Start method to make it
// spinning, and the Stop method to stop it and release background resources.
func newSpinner() *spinner {
	phases := []string{}
	for _, r := range "⠉⠘⠰⠤⠆⠃" {
		phases = append(phases, string(r)+" ")
	}
	s := &spinner{
		phases: phases,
		done:   make(chan struct{}),
	}
	return s
}

// Spinner returns the spinner string for the current phase.
func (s *spinner) Spinner() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.phases[s.phase]
}

// Start the spinner to spin in steps every specified interval.
func (s *spinner) Start(interval time.Duration) {
	s.ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.mu.Lock()
				s.phase++
				if s.phase >= len(s.phases) {
					s.phase = 0
				}
				s.mu.Unlock()
			case <-s.done:
				s.ticker.Stop()
				return
			}
		}
	}()
}

// Stop the spinner and release the background resources.
func (s *spinner) Stop() {
	close(s.done)
}
