package cmd

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newVolumesCmd(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "volumes",
		Short: "List available volumes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.json {
				return withShell(opts, func(sh *vbkshell.Shell) error {
					return printJSON(sh.VolumesInfo())
				})
			}
			return withShell(opts, func(sh *vbkshell.Shell) error {
				sh.Volumes()
				return nil
			})
		},
	}
}
