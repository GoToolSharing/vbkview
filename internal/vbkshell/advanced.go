package vbkshell

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	vbk "github.com/GoToolSharing/vbktoolkit"
)

type ExtractOptions struct {
	Resume       bool
	VerifySHA256 string
}

type CatResult struct {
	SourcePath   string `json:"source_path"`
	ResolvedPath string `json:"resolved_path"`
	ReadBytes    int    `json:"read_bytes"`
	TotalSize    uint64 `json:"total_size"`
	Truncated    bool   `json:"truncated"`
	Encoding     string `json:"encoding"`
	Content      string `json:"content"`
}

type StatInfo struct {
	Name          string         `json:"name"`
	Path          string         `json:"path"`
	Type          string         `json:"type"`
	IsDir         bool           `json:"is_dir"`
	IsFile        bool           `json:"is_file"`
	IsInternal    bool           `json:"is_internal_file"`
	IsExternal    bool           `json:"is_external_file"`
	SizeBytes     uint64         `json:"size_bytes"`
	SizeHuman     string         `json:"size_human"`
	PropertyCount int            `json:"property_count"`
	Properties    map[string]any `json:"properties,omitempty"`
}

type TreeNode struct {
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	IsDir     bool       `json:"is_dir"`
	SizeBytes uint64     `json:"size_bytes"`
	SizeHuman string     `json:"size_human"`
	Children  []TreeNode `json:"children,omitempty"`
}

type GrepOptions struct {
	IgnoreCase bool
	MaxBytes   int64
}

type GrepMatch struct {
	Path       string `json:"path"`
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
}

func (s *Shell) ExtractWithOptions(src, dst string, opts ExtractOptions) (GetResult, error) {
	if s.guest != nil {
		vols := s.guest.Volumes()
		if s.active < 0 || s.active >= len(vols) {
			return GetResult{}, fmt.Errorf("invalid active volume")
		}
		vol := vols[s.active]

		target := s.resolve(src)
		outPath := dst
		if strings.TrimSpace(outPath) == "" {
			outPath = path.Base(target)
		}
		if err := os.MkdirAll(path.Dir(outPath), 0o755); err != nil {
			return GetResult{}, err
		}

		res := GetResult{SourcePath: src, ResolvedPath: target, OutputPath: outPath}
		startOffset := int64(0)
		flags := os.O_CREATE | os.O_WRONLY
		if opts.Resume {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		out, err := os.OpenFile(outPath, flags, 0o644)
		if err != nil {
			return GetResult{}, err
		}
		defer out.Close()

		if opts.Resume {
			if st, statErr := out.Stat(); statErr == nil {
				if st.Size() > 0 {
					startOffset = st.Size()
					res.Resumed = true
				}
			}
		}

		written, err := vol.CopyFile(target, out, startOffset)
		if err != nil {
			return GetResult{}, err
		}
		res.BytesWritten = written

		if strings.TrimSpace(opts.VerifySHA256) != "" {
			sum, err := sha256File(outPath)
			if err != nil {
				return GetResult{}, err
			}
			expected := strings.ToLower(strings.TrimSpace(opts.VerifySHA256))
			res.SHA256 = sum
			res.SHA256Match = sum == expected
			if !res.SHA256Match {
				return res, fmt.Errorf("sha256 mismatch: expected %s got %s", expected, sum)
			}
		}

		return res, nil
	}

	target := s.resolve(src)
	item, err := s.v.Get(target, nil)
	if err != nil {
		return GetResult{}, err
	}
	if item.IsDir() {
		return GetResult{}, fmt.Errorf("%s is a directory", target)
	}

	size, err := item.Size()
	if err != nil {
		return GetResult{}, err
	}

	outPath := dst
	if strings.TrimSpace(outPath) == "" {
		outPath = path.Base(target)
	}
	if err := os.MkdirAll(path.Dir(outPath), 0o755); err != nil {
		return GetResult{}, err
	}

	in, err := item.Open()
	if err != nil {
		return GetResult{}, err
	}
	defer in.Close()

	res := GetResult{SourcePath: src, ResolvedPath: target, OutputPath: outPath}
	startOffset := int64(0)

	if opts.Resume {
		if st, statErr := os.Stat(outPath); statErr == nil {
			startOffset = st.Size()
			if uint64(startOffset) > size {
				return GetResult{}, fmt.Errorf("existing destination is larger than source file")
			}
			res.Resumed = startOffset > 0
		}
	}

	if startOffset > 0 {
		if _, err := in.Seek(startOffset, io.SeekStart); err != nil {
			return GetResult{}, err
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if opts.Resume {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	out, err := os.OpenFile(outPath, flags, 0o644)
	if err != nil {
		return GetResult{}, err
	}
	defer out.Close()

	written, err := io.CopyBuffer(out, in, make([]byte, 1024*1024))
	if err != nil {
		return GetResult{}, err
	}
	res.BytesWritten = written

	if strings.TrimSpace(opts.VerifySHA256) != "" {
		sum, err := sha256File(outPath)
		if err != nil {
			return GetResult{}, err
		}
		expected := strings.ToLower(strings.TrimSpace(opts.VerifySHA256))
		res.SHA256 = sum
		res.SHA256Match = sum == expected
		if !res.SHA256Match {
			return res, fmt.Errorf("sha256 mismatch: expected %s got %s", expected, sum)
		}
	}

	return res, nil
}

func (s *Shell) CatData(src string, limit int64, forceBase64 bool) (CatResult, error) {
	if s.guest != nil {
		vols := s.guest.Volumes()
		if s.active < 0 || s.active >= len(vols) {
			return CatResult{}, fmt.Errorf("invalid active volume")
		}
		target := s.resolve(src)
		size, err := vols[s.active].FileSize(target)
		if err != nil {
			return CatResult{}, err
		}
		data, err := vols[s.active].ReadFile(target, limit)
		if err != nil {
			return CatResult{}, err
		}

		res := CatResult{SourcePath: src, ResolvedPath: target, ReadBytes: len(data), TotalSize: size, Truncated: limit >= 0 && uint64(len(data)) < size}
		if forceBase64 || !utf8.Valid(data) {
			res.Encoding = "base64"
			res.Content = base64.StdEncoding.EncodeToString(data)
			return res, nil
		}
		res.Encoding = "utf-8"
		res.Content = string(data)
		return res, nil
	}

	target := s.resolve(src)
	item, err := s.v.Get(target, nil)
	if err != nil {
		return CatResult{}, err
	}
	if item.IsDir() {
		return CatResult{}, fmt.Errorf("%s is a directory", target)
	}

	size, _ := item.Size()
	stream, err := item.Open()
	if err != nil {
		return CatResult{}, err
	}
	defer stream.Close()

	var data []byte
	if limit >= 0 {
		data, err = io.ReadAll(io.LimitReader(stream, limit))
	} else {
		data, err = io.ReadAll(stream)
	}
	if err != nil {
		return CatResult{}, err
	}

	res := CatResult{
		SourcePath:   src,
		ResolvedPath: target,
		ReadBytes:    len(data),
		TotalSize:    size,
		Truncated:    limit >= 0 && uint64(len(data)) < size,
	}

	if forceBase64 || !utf8.Valid(data) {
		res.Encoding = "base64"
		res.Content = base64.StdEncoding.EncodeToString(data)
		return res, nil
	}
	res.Encoding = "utf-8"
	res.Content = string(data)
	return res, nil
}

func (s *Shell) Stat(pathArg string, includeProps bool) (StatInfo, error) {
	if s.guest != nil {
		vols := s.guest.Volumes()
		if s.active < 0 || s.active >= len(vols) {
			return StatInfo{}, fmt.Errorf("invalid active volume")
		}
		target := s.cwd
		if strings.TrimSpace(pathArg) != "" {
			target = s.resolve(pathArg)
		}
		isDir, err := vols[s.active].IsDir(target)
		if err != nil {
			return StatInfo{}, err
		}
		size := uint64(0)
		if !isDir {
			size, _ = vols[s.active].FileSize(target)
		}
		return StatInfo{
			Name:          path.Base(target),
			Path:          target,
			Type:          map[bool]string{true: "directory", false: "file"}[isDir],
			IsDir:         isDir,
			IsFile:        !isDir,
			IsInternal:    !isDir,
			IsExternal:    false,
			SizeBytes:     size,
			SizeHuman:     humanSize(size),
			PropertyCount: 0,
			Properties:    nil,
		}, nil
	}

	target := s.cwd
	if strings.TrimSpace(pathArg) != "" {
		target = s.resolve(pathArg)
	}

	item, err := s.v.Get(target, nil)
	if err != nil {
		return StatInfo{}, err
	}

	size := uint64(0)
	if item.IsFile() {
		size, _ = item.Size()
	}

	res := StatInfo{
		Name:       item.Name,
		Path:       target,
		Type:       dirItemTypeToString(item),
		IsDir:      item.IsDir(),
		IsFile:     item.IsFile(),
		IsInternal: item.IsInternalFile(),
		IsExternal: item.IsExternalFile(),
		SizeBytes:  size,
		SizeHuman:  humanSize(size),
	}

	props, err := item.Properties()
	if err != nil {
		return StatInfo{}, err
	}
	res.PropertyCount = len(props)
	if includeProps && len(props) > 0 {
		res.Properties = make(map[string]any, len(props))
		for k, v := range props {
			res.Properties[k] = v
		}
	}

	return res, nil
}

func (s *Shell) Tree(pathArg string, maxDepth int) (TreeNode, error) {
	if s.guest != nil {
		vols := s.guest.Volumes()
		if s.active < 0 || s.active >= len(vols) {
			return TreeNode{}, fmt.Errorf("invalid active volume")
		}
		start := s.cwd
		if strings.TrimSpace(pathArg) != "" {
			start = s.resolve(pathArg)
		}
		return buildGuestTree(vols[s.active], start, maxDepth, 0)
	}

	target := s.cwd
	if strings.TrimSpace(pathArg) != "" {
		target = s.resolve(pathArg)
	}

	item, err := s.v.Get(target, nil)
	if err != nil {
		return TreeNode{}, err
	}

	return s.treeNode(target, item, maxDepth, 0)
}

func (s *Shell) Grep(pattern, start string, opts GrepOptions) ([]GrepMatch, error) {
	if s.guest != nil {
		vols := s.guest.Volumes()
		if s.active < 0 || s.active >= len(vols) {
			return nil, fmt.Errorf("invalid active volume")
		}
		root := s.cwd
		if strings.TrimSpace(start) != "" {
			root = s.resolve(start)
		}

		re := pattern
		if opts.IgnoreCase {
			re = "(?i)" + re
		}
		rx, err := regexp.Compile(re)
		if err != nil {
			return nil, err
		}

		var matches []GrepMatch
		var walk func(string) error
		walk = func(cur string) error {
			entries, err := vols[s.active].ListDir(cur)
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.IsDir {
					_ = walk(e.Path)
					continue
				}
				data, err := vols[s.active].ReadFile(e.Path, opts.MaxBytes)
				if err != nil {
					continue
				}
				scanner := bufio.NewScanner(strings.NewReader(string(data)))
				lineNo := 0
				for scanner.Scan() {
					lineNo++
					line := scanner.Text()
					if rx.MatchString(line) {
						matches = append(matches, GrepMatch{Path: e.Path, LineNumber: lineNo, Line: line})
					}
				}
			}
			return nil
		}
		if err := walk(root); err != nil {
			return nil, err
		}
		return matches, nil
	}

	root := s.cwd
	if strings.TrimSpace(start) != "" {
		root = s.resolve(start)
	}

	re := pattern
	if opts.IgnoreCase {
		re = "(?i)" + re
	}
	rx, err := regexp.Compile(re)
	if err != nil {
		return nil, err
	}

	results := make([]GrepMatch, 0, 16)
	err = s.walk(root, func(p string, item *vbk.DirItem) error {
		if item.IsDir() {
			return nil
		}

		stream, err := item.Open()
		if err != nil {
			return nil
		}
		defer stream.Close()

		reader := io.Reader(stream)
		if opts.MaxBytes > 0 {
			reader = io.LimitReader(stream, opts.MaxBytes)
		}

		scanner := bufio.NewScanner(reader)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if rx.MatchString(line) {
				results = append(results, GrepMatch{Path: p, LineNumber: lineNo, Line: line})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Path == results[j].Path {
			return results[i].LineNumber < results[j].LineNumber
		}
		return results[i].Path < results[j].Path
	})

	return results, nil
}

func (s *Shell) treeNode(curPath string, item *vbk.DirItem, maxDepth, depth int) (TreeNode, error) {
	size := uint64(0)
	if item.IsFile() {
		size, _ = item.Size()
	}

	node := TreeNode{
		Name:      item.Name,
		Path:      curPath,
		IsDir:     item.IsDir(),
		SizeBytes: size,
		SizeHuman: humanSize(size),
	}

	if !item.IsDir() {
		return node, nil
	}
	if maxDepth >= 0 && depth >= maxDepth {
		return node, nil
	}

	children, err := item.IterDir()
	if err != nil {
		return TreeNode{}, err
	}
	sort.Slice(children, func(i, j int) bool {
		if children[i].IsDir() != children[j].IsDir() {
			return children[i].IsDir()
		}
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})

	node.Children = make([]TreeNode, 0, len(children))
	for _, child := range children {
		nextPath := joinPath(curPath, child.Name)
		nextNode, err := s.treeNode(nextPath, child, maxDepth, depth+1)
		if err != nil {
			return TreeNode{}, err
		}
		node.Children = append(node.Children, nextNode)
	}

	return node, nil
}

func sha256File(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func dirItemTypeToString(item *vbk.DirItem) string {
	if item.IsDir() {
		return "directory"
	}
	if item.IsInternalFile() {
		return "internal_file"
	}
	if item.IsExternalFile() {
		return "external_file"
	}
	return "file"
}

func buildGuestTree(vol *vbk.GuestVolume, curPath string, maxDepth, depth int) (TreeNode, error) {
	isDir, err := vol.IsDir(curPath)
	if err != nil {
		return TreeNode{}, err
	}
	size := uint64(0)
	if !isDir {
		size, _ = vol.FileSize(curPath)
	}

	node := TreeNode{
		Name:      path.Base(curPath),
		Path:      curPath,
		IsDir:     isDir,
		SizeBytes: size,
		SizeHuman: humanSize(size),
	}
	if node.Name == "" || node.Name == "." {
		node.Name = "/"
	}

	if !isDir || (maxDepth >= 0 && depth >= maxDepth) {
		return node, nil
	}

	entries, err := vol.ListDir(curPath)
	if err != nil {
		return TreeNode{}, err
	}
	node.Children = make([]TreeNode, 0, len(entries))
	for _, e := range entries {
		child, err := buildGuestTree(vol, e.Path, maxDepth, depth+1)
		if err != nil {
			continue
		}
		node.Children = append(node.Children, child)
	}
	return node, nil
}
