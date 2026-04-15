package cmd

import (
	"fmt"

	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newPwdCmd(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "pwd",
		Short: "Print current working directory in VBK",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.json {
				return withShell(opts, func(sh *vbkshell.Shell) error {
					return printJSON(map[string]string{"cwd": sh.PWD()})
				})
			}
			return withShell(opts, func(sh *vbkshell.Shell) error {
				fmt.Println(sh.PWD())
				return nil
			})
		},
	}
}
