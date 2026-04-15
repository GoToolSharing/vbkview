package cmd

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newShellCmd(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Start interactive VBK shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withShell(opts, func(sh *vbkshell.Shell) error {
				return sh.Run()
			})
		},
	}
}
