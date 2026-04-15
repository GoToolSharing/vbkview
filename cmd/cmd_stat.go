package cmd

import (
	"sort"

	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newStatCmd(opts *globalOptions) *cobra.Command {
	var includeProps bool

	cmd := &cobra.Command{
		Use:   "stat [path]",
		Short: "Show metadata for a VBK path",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathArg := ""
			if len(args) == 1 {
				pathArg = args[0]
			}

			return withShell(opts, func(sh *vbkshell.Shell) error {
				st, err := sh.Stat(pathArg, includeProps)
				if err != nil {
					return err
				}

				if opts.json {
					return printJSON(st)
				}

				cmd.Printf("Path: %s\n", st.Path)
				cmd.Printf("Type: %s\n", st.Type)
				cmd.Printf("Size: %s (%d bytes)\n", st.SizeHuman, st.SizeBytes)
				cmd.Printf("Properties: %d\n", st.PropertyCount)
				if includeProps && len(st.Properties) > 0 {
					keys := make([]string, 0, len(st.Properties))
					for k := range st.Properties {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					for _, k := range keys {
						v := st.Properties[k]
						cmd.Printf("  %s=%v\n", k, v)
					}
				}
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&includeProps, "props", true, "include VBK properties when available")
	return cmd
}
