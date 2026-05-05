package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	vbk "github.com/GoToolSharing/vbktoolkit"
	"github.com/GoToolSharing/vbkview/internal/vbkshell"
	"github.com/spf13/cobra"
)

type getDiskResult struct {
	SourcePath   string `json:"source_path"`
	OutputPath   string `json:"output_path"`
	BytesWritten int64  `json:"bytes_written"`
	DiskSize     uint64 `json:"disk_size_bytes"`
	SHA256       string `json:"sha256,omitempty"`
}

func newGetDiskCmd(opts *globalOptions) *cobra.Command {
	var sha256Expected string

	cmd := &cobra.Command{
		Use:   "get-disk <src> [dst]",
		Short: "Extract a virtual disk from VBK as a flat raw image",
		Long: `Extract a .vmdk, .vhd, or .vhdx virtual disk from a VBK backup as a flat
raw disk image. Multi-extent VMDKs are reassembled automatically.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.vbkPath == "" {
				return fmt.Errorf("--vbk is required")
			}

			srcArg := args[0]
			dstArg := ""
			if len(args) == 2 {
				dstArg = args[1]
			}

			v, fh, err := vbk.Open(opts.vbkPath, opts.verify)
			if err != nil {
				return err
			}
			defer fh.Close()

			src := vbkshell.NormalizePath(srcArg, opts.cwd)

			img, err := v.OpenDiskImage(src)
			if err != nil {
				return err
			}
			defer img.Close()

			diskSize := img.Size()

			outPath := dstArg
			if strings.TrimSpace(outPath) == "" {
				outPath = path.Base(src)
			}
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}

			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}

			h := sha256.New()
			var writer io.Writer = out
			if strings.TrimSpace(sha256Expected) != "" {
				writer = io.MultiWriter(out, h)
			}

			written, copyErr := io.CopyBuffer(writer, img, make([]byte, 1<<20))
			out.Close()
			if copyErr != nil {
				os.Remove(outPath)
				return copyErr
			}

			if uint64(written) != diskSize {
				os.Remove(outPath)
				return fmt.Errorf("extraction incomplete: wrote %d of %d bytes", written, diskSize)
			}

			var actualSum string
			if strings.TrimSpace(sha256Expected) != "" {
				actualSum = hex.EncodeToString(h.Sum(nil))
				expected := strings.ToLower(strings.TrimSpace(sha256Expected))
				if actualSum != expected {
					os.Remove(outPath)
					return fmt.Errorf("sha256 mismatch: expected %s got %s", expected, actualSum)
				}
			}

			res := getDiskResult{
				SourcePath:   src,
				OutputPath:   outPath,
				BytesWritten: written,
				DiskSize:     diskSize,
				SHA256:       actualSum,
			}

			if opts.json {
				return printJSON(res)
			}

			cmd.Printf("Saved to %s (%d bytes)\n", res.OutputPath, res.BytesWritten)
			if res.SHA256 != "" {
				cmd.Printf("SHA256 OK (%s)\n", res.SHA256)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sha256Expected, "sha256", "", "verify extracted disk SHA-256")
	return cmd
}
