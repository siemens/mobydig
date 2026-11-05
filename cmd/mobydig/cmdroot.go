// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	rtdebug "runtime/debug"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	indentation     *uint
	spinnerInterval *time.Duration
	workerNumber    *uint
	debug           *bool
)

func newRootCmd() (rootCmd *cobra.Command) {
	rootCmd = &cobra.Command{
		Use:     "mobydig [flags] containername",
		Short:   "mobydig digs and validates DNS names on all networks attached to a specific container",
		Version: version(),
		Args:    cobra.ExactArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if *indentation > 80 {
				return fmt.Errorf("--indentation width out of range [0..80]")
			}
			if *workerNumber < 1 || *workerNumber > 10 {
				return fmt.Errorf("--workers out of range [1..10]")
			}
			if *spinnerInterval < 10*time.Millisecond {
				return fmt.Errorf("--spinner must be at least 10ms")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// from here on, errors aren't about using the command incorrectly.
			cmd.SilenceUsage = true
			if *debug {
				slog.SetDefault(slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelDebug})))
				slog.Debug("debug logging enabled")
			}
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return DigAndReport(ctx, cmd.OutOrStdout(), args[0])
		},
	}
	// Sets up the flags.
	debug = rootCmd.PersistentFlags().Bool(
		"debug", false, "enable debugging output")
	indentation = rootCmd.PersistentFlags().Uint(
		"indent", 3, "indentation width")
	spinnerInterval = rootCmd.PersistentFlags().Duration(
		"spinner", 100*time.Millisecond, "spinner interval")
	workerNumber = rootCmd.PersistentFlags().Uint(
		"workers", 5, "number of DNS and ping workers")
	return
}

// version returns the module version mobydig was built from.
func version() string {
	if bi, ok := rtdebug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}
