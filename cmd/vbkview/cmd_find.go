package main

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newFindCmd(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "find <name> [start]",
		Short: "Find files by name in VBK",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := ""
			if len(args) == 2 {
				start = args[1]
			}
			if opts.json {
				return withShell(opts, func(sh *vbkshell.Shell) error {
					matches, err := sh.FindMatches(args[0], start)
					if err != nil {
						return err
					}
					return printJSON(matches)
				})
			}
			return withShell(opts, func(sh *vbkshell.Shell) error {
				return sh.Find(args[0], start)
			})
		},
	}
}
