package cmd

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newDisksCmd(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "disks",
		Short: "List .vhd/.vhdx entries found in VBK",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.json {
				return withShell(opts, func(sh *vbkshell.Shell) error {
					return printJSON(sh.DisksList())
				})
			}
			return withShell(opts, func(sh *vbkshell.Shell) error {
				sh.Disks()
				return nil
			})
		},
	}
}
