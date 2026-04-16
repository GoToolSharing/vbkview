package vbkshell

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"

	vbk "github.com/GoToolSharing/vbktoolkit"
	"github.com/GoToolSharing/vbkview/internal/cli"
	"github.com/peterh/liner"
)

type Shell struct {
	vbkPath string
	fh      *os.File
	v       *vbk.VBK
	cwd     string
	disks   []string
	guest   *vbk.Guest
	active  int
}

func (s *Shell) PWD() string {
	return s.cwd
}

func (s *Shell) CD(p string) error {
	return s.cmdCD(p)
}

func (s *Shell) LS(p string, long bool) error {
	return s.cmdLS(p, long)
}

func (s *Shell) Cat(p string, limit int64) error {
	return s.cmdCat(p, limit)
}

func (s *Shell) Get(src, dst string) error {
	return s.cmdGet(src, dst)
}

func (s *Shell) GetWithOptions(src, dst string, opts ExtractOptions) (GetResult, error) {
	return s.ExtractWithOptions(src, dst, opts)
}

func (s *Shell) Find(name, start string) error {
	return s.cmdFind(name, start)
}

func (s *Shell) Disks() {
	s.cmdDisks()
}

func (s *Shell) Volumes() {
	s.cmdVolumes()
}

func New(vbkPath string, verify bool) (*Shell, error) {
	v, fh, err := vbk.Open(vbkPath, verify)
	if err != nil {
		return nil, err
	}

	sh := &Shell{vbkPath: vbkPath, fh: fh, v: v, cwd: "/", active: 0}
	sh.disks, _ = sh.findVirtualDisks()
	if guest, err := v.DiscoverGuest(); err == nil {
		if len(guest.Volumes()) > 0 {
			sh.guest = guest
			sh.active = guest.DefaultIndex()
		} else {
			_ = guest.Close()
		}
	}
	return sh, nil
}

func (s *Shell) Close() {
	if s.guest != nil {
		_ = s.guest.Close()
	}
	if s.fh != nil {
		_ = s.fh.Close()
	}
}

func (s *Shell) Run() error {
	s.cmdVolumes()
	if len(s.disks) > 0 {
		fmt.Printf("Virtual disks detected: %d (run `disks`)\n", len(s.disks))
	}
	s.cmdHelp()

	lineReader := liner.NewLiner()
	defer lineReader.Close()
	lineReader.SetCtrlCAborts(true)
	lineReader.SetCompleter(s.completeInput)

	for {
		line, err := lineReader.Prompt(s.prompt())
		if errors.Is(err, io.EOF) {
			fmt.Println()
			return nil
		}
		if isPromptInterrupted(err) {
			fmt.Println()
			continue
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lineReader.AppendHistory(line)

		parts, err := cli.Split(line)
		if err != nil {
			fmt.Printf("Parse error: %v\n", err)
			continue
		}
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "exit", "quit":
			return nil
		case "help":
			s.cmdHelp()
		case "volumes":
			s.cmdVolumes()
		case "use":
			if len(args) != 1 {
				fmt.Println("Usage: use <idx>")
				continue
			}
			if err := s.cmdUse(args[0]); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "pwd":
			s.cmdPWD()
		case "ls":
			var p string
			if len(args) > 0 {
				p = args[0]
			}
			if err := s.cmdLS(p, false); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "ll":
			var p string
			if len(args) > 0 {
				p = args[0]
			}
			if err := s.cmdLS(p, true); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "cd":
			if len(args) != 1 {
				fmt.Println("Usage: cd <path>")
				continue
			}
			if err := s.cmdCD(args[0]); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "cat":
			if len(args) != 1 {
				fmt.Println("Usage: cat <file>")
				continue
			}
			if err := s.cmdCat(args[0], -1); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "get":
			if len(args) < 1 || len(args) > 2 {
				fmt.Println("Usage: get <src> [dst]")
				continue
			}
			var dst string
			if len(args) == 2 {
				dst = args[1]
			}
			if err := s.cmdGet(args[0], dst); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "find":
			if len(args) < 1 || len(args) > 2 {
				fmt.Println("Usage: find <name> [start]")
				continue
			}
			var start string
			if len(args) == 2 {
				start = args[1]
			}
			if err := s.cmdFind(args[0], start); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "stat":
			var p string
			if len(args) > 0 {
				p = args[0]
			}
			if err := s.cmdStat(p); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "tree":
			var p string
			depth := -1
			if len(args) > 0 {
				p = args[0]
			}
			if len(args) > 1 {
				d, err := strconv.Atoi(args[1])
				if err != nil {
					fmt.Printf("Error: invalid depth: %v\n", err)
					continue
				}
				depth = d
			}
			if err := s.cmdTree(p, depth); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "grep":
			if len(args) < 1 || len(args) > 2 {
				fmt.Println("Usage: grep <pattern> [start]")
				continue
			}
			start := ""
			if len(args) == 2 {
				start = args[1]
			}
			if err := s.cmdGrep(args[0], start); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "disks":
			s.cmdDisks()
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}

func isPromptInterrupted(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, liner.ErrPromptAborted) {
		return true
	}
	if errors.Is(err, syscall.EINTR) || errors.Is(err, fs.ErrClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "interrupt") || strings.Contains(msg, "aborted")
}

func (s *Shell) resolve(p string) string {
	return normalizePath(p, s.cwd)
}

func (s *Shell) cmdVolumes() {
	for _, v := range s.VolumesInfo() {
		marker := " "
		if v.Index == s.active {
			marker = "*"
		}
		if s.guest != nil {
			fmt.Printf("%s [%d] disk=%s volume=%d name=%q size=%s\n", marker, v.Index, v.Source, v.VolumeIndex, v.Name, v.Size)
			continue
		}
		fmt.Printf("%s [%d] source=%q name=%q size=%s\n", marker, v.Index, v.Source, v.Name, v.Size)
	}
}

func (s *Shell) cmdUse(idx string) error {
	i, err := strconv.Atoi(idx)
	if err != nil {
		return err
	}
	if s.guest != nil {
		if i < 0 || i >= len(s.guest.Volumes()) {
			return fmt.Errorf("invalid volume index: %d", i)
		}
		s.active = i
		s.cwd = "/"
		return nil
	}
	if i != 0 {
		return fmt.Errorf("invalid volume index: %d (only 0 is available)", i)
	}
	s.cwd = "/"
	return nil
}

func (s *Shell) cmdPWD() {
	fmt.Println(s.cwd)
}

func (s *Shell) cmdLS(p string, long bool) error {
	entries, err := s.LSEntries(p)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if long {
			kind := "-"
			if e.IsDir {
				kind = "d"
			}
			fmt.Printf("%s %8s %s\n", kind, e.SizeHuman, e.Name)
		} else {
			suffix := ""
			if e.IsDir {
				suffix = "/"
			}
			fmt.Printf("%s%s\n", e.Name, suffix)
		}
	}

	return nil
}

func (s *Shell) cmdCD(p string) error {
	if s.guest != nil {
		vols := s.guest.Volumes()
		if s.active < 0 || s.active >= len(vols) {
			return fmt.Errorf("invalid active volume")
		}
		target := s.resolve(p)
		isDir, err := vols[s.active].IsDir(target)
		if err != nil {
			return err
		}
		if !isDir {
			return fmt.Errorf("%s is not a directory", target)
		}
		s.cwd = target
		return nil
	}

	target := s.resolve(p)
	item, err := s.v.Get(target, nil)
	if err != nil {
		return err
	}
	if !item.IsDir() {
		return fmt.Errorf("%s is not a directory", target)
	}
	s.cwd = target
	return nil
}

func (s *Shell) cmdCat(p string, limit int64) error {
	res, err := s.CatData(p, limit, false)
	if err != nil {
		return err
	}
	fmt.Println(res.Content)
	return nil
}

func (s *Shell) cmdGet(src, dst string) error {
	res, err := s.ExtractWithOptions(src, dst, ExtractOptions{})
	if err != nil {
		return err
	}
	if res.Resumed {
		fmt.Printf("Saved to %s (%d bytes appended)\n", res.OutputPath, res.BytesWritten)
		return nil
	}
	fmt.Printf("Saved to %s (%d bytes)\n", res.OutputPath, res.BytesWritten)
	return nil
}

func (s *Shell) cmdFind(name, start string) error {
	paths, err := s.FindMatches(name, start)
	if err != nil {
		return err
	}
	for _, p := range paths {
		fmt.Println(p)
	}
	return nil
}

func (s *Shell) cmdDisks() {
	disks := s.DisksList()
	if len(disks) == 0 {
		fmt.Println("No .vhd/.vhdx/.vmdk entries found in this VBK")
		return
	}
	for i, d := range disks {
		fmt.Printf("[%d] %s\n", i, d)
	}
}

func (s *Shell) cmdHelp() {
	fmt.Print(
		"Commands:\n" +
			"  volumes                  List available volumes\n" +
			"  use <idx>                Change active volume\n" +
			"  disks                    List .vhd/.vhdx/.vmdk entries in VBK\n" +
			"  pwd                      Print current directory\n" +
			"  ls [path]                List directory\n" +
			"  ll [path]                List directory (long)\n" +
			"  cd <path>                Change directory\n" +
			"  cat <file>               Print text file\n" +
			"  get <src> [dst]          Extract file to local disk\n" +
			"  find <name> [start]      Find file by name\n" +
			"  stat [path]              Show metadata for a path\n" +
			"  tree [path] [depth]      Print directory tree\n" +
			"  grep <pattern> [start]   Search text in files\n" +
			"  help                     Show help\n" +
			"  exit | quit              Quit\n",
	)
}

func (s *Shell) cmdStat(pathArg string) error {
	st, err := s.Stat(pathArg, true)
	if err != nil {
		return err
	}
	fmt.Printf("Path: %s\n", st.Path)
	fmt.Printf("Type: %s\n", st.Type)
	fmt.Printf("Size: %s (%d bytes)\n", st.SizeHuman, st.SizeBytes)
	fmt.Printf("Properties: %d\n", st.PropertyCount)
	if st.PropertyCount > 0 {
		keys := make([]string, 0, len(st.Properties))
		for k := range st.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %s=%v\n", k, st.Properties[k])
		}
	}
	return nil
}

func (s *Shell) cmdTree(pathArg string, depth int) error {
	node, err := s.Tree(pathArg, depth)
	if err != nil {
		return err
	}
	printTreeNode(node, "", true)
	return nil
}

func (s *Shell) cmdGrep(pattern, start string) error {
	matches, err := s.Grep(pattern, start, GrepOptions{MaxBytes: 2 * 1024 * 1024})
	if err != nil {
		return err
	}
	for _, m := range matches {
		fmt.Printf("%s:%d:%s\n", m.Path, m.LineNumber, m.Line)
	}
	return nil
}

func printTreeNode(node TreeNode, prefix string, isLast bool) {
	name := node.Name
	if strings.TrimSpace(name) == "" {
		name = "/"
	}

	if prefix == "" {
		fmt.Println(name)
	} else {
		connector := "|-- "
		if isLast {
			connector = "`-- "
		}
		suffix := ""
		if node.IsDir {
			suffix = "/"
		}
		fmt.Printf("%s%s%s%s\n", prefix, connector, name, suffix)
	}

	nextPrefix := prefix
	if prefix != "" {
		if isLast {
			nextPrefix += "    "
		} else {
			nextPrefix += "|   "
		}
	}

	for i := range node.Children {
		printTreeNode(node.Children[i], nextPrefix, i == len(node.Children)-1)
	}
}

func (s *Shell) findVirtualDisks() ([]string, error) {
	out := make([]string, 0, 4)
	err := s.walk("/", func(p string, item *vbk.DirItem) error {
		if item.IsDir() {
			return nil
		}
		low := strings.ToLower(item.Name)
		if strings.HasSuffix(low, ".vhd") || strings.HasSuffix(low, ".vhdx") || strings.HasSuffix(low, ".vmdk") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Shell) walk(root string, fn func(p string, item *vbk.DirItem) error) error {
	start, err := s.v.Get(root, nil)
	if err != nil {
		return err
	}

	var walkRec func(curPath string, item *vbk.DirItem) error
	walkRec = func(curPath string, item *vbk.DirItem) error {
		if err := fn(curPath, item); err != nil {
			return err
		}
		if !item.IsDir() {
			return nil
		}

		entries, err := item.IterDir()
		if err != nil {
			return err
		}
		for _, child := range entries {
			next := path.Join(curPath, child.Name)
			if !strings.HasPrefix(next, "/") {
				next = "/" + next
			}
			if err := walkRec(next, child); err != nil {
				return err
			}
		}
		return nil
	}

	return walkRec(normalizePath(root, "/"), start)
}

func (s *Shell) prompt() string {
	if s.guest != nil {
		vols := s.guest.Volumes()
		if s.active >= 0 && s.active < len(vols) {
			return fmt.Sprintf("vbk[%d:%s %s]> ", s.active, vols[s.active].Name, s.cwd)
		}
	}
	return fmt.Sprintf("vbk[%s]> ", s.cwd)
}
