package cmd

import (
	"fmt"

	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

type globalOptions struct {
	vbkPath string
	verify  bool
	cwd     string
	json    bool
}

func NewRootCmd() *cobra.Command {
	opts := &globalOptions{}

	root := &cobra.Command{
		Use:   "vbkview",
		Short: "Inspect and extract content from VBK backups",
		Long:  "vbkview provides interactive and non-interactive commands to inspect VBK backup content.",
	}

	root.PersistentFlags().StringVarP(&opts.vbkPath, "vbk", "f", "", "path to .vbk file")
	root.PersistentFlags().BoolVar(&opts.verify, "verify", true, "verify VBK metadata CRCs")
	root.PersistentFlags().StringVar(&opts.cwd, "cwd", "/", "working directory inside VBK for relative paths")
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "output in JSON when supported")

	root.AddCommand(newShellCmd(opts))
	root.AddCommand(newPwdCmd(opts))
	root.AddCommand(newLsCmd(opts))
	root.AddCommand(newCatCmd(opts))
	root.AddCommand(newGetCmd(opts))
	root.AddCommand(newGetDiskCmd(opts))
	root.AddCommand(newFindCmd(opts))
	root.AddCommand(newStatCmd(opts))
	root.AddCommand(newTreeCmd(opts))
	root.AddCommand(newGrepCmd(opts))
	root.AddCommand(newDisksCmd(opts))
	root.AddCommand(newVolumesCmd(opts))

	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.SetVersionTemplate("{{.Version}}\n")

	return root
}

func withShell(opts *globalOptions, fn func(*vbkshell.Shell) error) error {
	if opts.vbkPath == "" {
		return fmt.Errorf("--vbk is required")
	}

	sh, err := vbkshell.New(opts.vbkPath, opts.verify)
	if err != nil {
		return err
	}
	defer sh.Close()

	if opts.cwd != "" && opts.cwd != "/" {
		if err := sh.CD(opts.cwd); err != nil {
			return err
		}
	}

	return fn(sh)
}
