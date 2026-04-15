package main

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newGetCmd(opts *globalOptions) *cobra.Command {
	var resume bool
	var sha256 string

	cmd := &cobra.Command{
		Use:   "get <src> [dst]",
		Short: "Extract a file from VBK to local disk",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dst := ""
			if len(args) == 2 {
				dst = args[1]
			}

			extractOpts := vbkshell.ExtractOptions{Resume: resume, VerifySHA256: sha256}

			return withShell(opts, func(sh *vbkshell.Shell) error {
				res, err := sh.GetWithOptions(args[0], dst, extractOpts)
				if err != nil {
					return err
				}

				if opts.json {
					return printJSON(res)
				}

				if res.Resumed {
					cmd.Printf("Saved to %s (%d bytes appended)\n", res.OutputPath, res.BytesWritten)
				} else {
					cmd.Printf("Saved to %s (%d bytes)\n", res.OutputPath, res.BytesWritten)
				}
				if res.SHA256 != "" {
					cmd.Printf("SHA256: %s\n", res.SHA256)
				}
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&resume, "resume", false, "resume writing if destination file already exists")
	cmd.Flags().StringVar(&sha256, "sha256", "", "verify extracted file SHA-256")
	return cmd
}
