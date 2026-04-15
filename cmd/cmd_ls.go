package cmd

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newLsCmd(opts *globalOptions) *cobra.Command {
	var long bool

	cmd := &cobra.Command{
		Use:   "ls [path]",
		Short: "List VBK directory content",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := ""
			if len(args) == 1 {
				p = args[0]
			}
			if opts.json {
				return withShell(opts, func(sh *vbkshell.Shell) error {
					entries, err := sh.LSEntries(p)
					if err != nil {
						return err
					}
					return printJSON(entries)
				})
			}
			return withShell(opts, func(sh *vbkshell.Shell) error {
				return sh.LS(p, long)
			})
		},
	}

	cmd.Flags().BoolVarP(&long, "long", "l", false, "long listing format")
	return cmd
}
