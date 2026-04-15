package main

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newCatCmd(opts *globalOptions) *cobra.Command {
	var limit int64
	var base64Output bool

	cmd := &cobra.Command{
		Use:   "cat <path>",
		Short: "Print a file from VBK",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withShell(opts, func(sh *vbkshell.Shell) error {
				res, err := sh.CatData(args[0], limit, base64Output)
				if err != nil {
					return err
				}

				if opts.json {
					return printJSON(res)
				}

				cmd.Println(res.Content)
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&limit, "limit", -1, "maximum bytes to read")
	cmd.Flags().BoolVar(&base64Output, "base64", false, "force base64 output")
	return cmd
}
