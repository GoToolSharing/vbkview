package cmd

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newGrepCmd(opts *globalOptions) *cobra.Command {
	var ignoreCase bool
	var maxBytes int64

	cmd := &cobra.Command{
		Use:   "grep <pattern> [start]",
		Short: "Search text in VBK files",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := ""
			if len(args) == 2 {
				start = args[1]
			}

			grepOpts := vbkshell.GrepOptions{IgnoreCase: ignoreCase, MaxBytes: maxBytes}
			return withShell(opts, func(sh *vbkshell.Shell) error {
				matches, err := sh.Grep(args[0], start, grepOpts)
				if err != nil {
					return err
				}

				if opts.json {
					return printJSON(matches)
				}

				for _, m := range matches {
					cmd.Printf("%s:%d:%s\n", m.Path, m.LineNumber, m.Line)
				}
				return nil
			})
		},
	}

	cmd.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "ignore letter case")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", 2*1024*1024, "max bytes read per file (0 means unlimited)")
	return cmd
}
