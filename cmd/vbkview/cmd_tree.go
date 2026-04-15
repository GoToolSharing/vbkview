package main

import (
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

func newTreeCmd(opts *globalOptions) *cobra.Command {
	var depth int

	cmd := &cobra.Command{
		Use:   "tree [path]",
		Short: "Print VBK directory tree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathArg := ""
			if len(args) == 1 {
				pathArg = args[0]
			}

			return withShell(opts, func(sh *vbkshell.Shell) error {
				node, err := sh.Tree(pathArg, depth)
				if err != nil {
					return err
				}

				if opts.json {
					return printJSON(node)
				}

				printTree(node, cmd)
				return nil
			})
		},
	}

	cmd.Flags().IntVar(&depth, "depth", -1, "maximum depth (-1 for unlimited)")
	return cmd
}

func printTree(node vbkshell.TreeNode, cmd *cobra.Command) {
	cmd.Println(node.Name)
	for i := range node.Children {
		printTreeNodeRecursive(node.Children[i], "", i == len(node.Children)-1, cmd)
	}
}

func printTreeNodeRecursive(node vbkshell.TreeNode, prefix string, isLast bool, cmd *cobra.Command) {
	connector := "|-- "
	if isLast {
		connector = "`-- "
	}
	suffix := ""
	if node.IsDir {
		suffix = "/"
	}
	cmd.Printf("%s%s%s%s\n", prefix, connector, node.Name, suffix)

	nextPrefix := prefix
	if isLast {
		nextPrefix += "    "
	} else {
		nextPrefix += "|   "
	}

	for i := range node.Children {
		printTreeNodeRecursive(node.Children[i], nextPrefix, i == len(node.Children)-1, cmd)
	}
}
