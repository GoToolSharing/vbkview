package vbkshell

import (
	"fmt"
	"sort"
	"strings"

	vbk "github.com/GoToolSharing/vbktoolkit"
)

type VolumeInfo struct {
	Index       int    `json:"index"`
	Source      string `json:"source"`
	Name        string `json:"name"`
	Size        string `json:"size"`
	VolumeIndex int    `json:"volume_index,omitempty"`
}

type EntryInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	SizeBytes uint64 `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
}

type GetResult struct {
	SourcePath   string `json:"source_path"`
	ResolvedPath string `json:"resolved_path"`
	OutputPath   string `json:"output_path"`
	BytesWritten int64  `json:"bytes_written"`
	Resumed      bool   `json:"resumed"`
	SHA256       string `json:"sha256,omitempty"`
	SHA256Match  bool   `json:"sha256_match,omitempty"`
}

func (s *Shell) VolumesInfo() []VolumeInfo {
	if s.guest != nil {
		volumes := s.guest.Volumes()
		out := make([]VolumeInfo, 0, len(volumes))
		for _, vol := range volumes {
			out = append(out, VolumeInfo{
				Index:       vol.Index,
				Source:      vol.DiskPath,
				Name:        vol.Name,
				Size:        humanSize(vol.Size),
				VolumeIndex: vol.VolumeIndex,
			})
		}
		return out
	}

	return []VolumeInfo{{
		Index:  0,
		Source: s.vbkPath,
		Name:   "vbk-root",
		Size:   "n/a",
	}}
}

func (s *Shell) DisksList() []string {
	out := make([]string, len(s.disks))
	copy(out, s.disks)
	return out
}

func (s *Shell) LSEntries(p string) ([]EntryInfo, error) {
	if s.guest != nil {
		if s.active < 0 || s.active >= len(s.guest.Volumes()) {
			return nil, fmt.Errorf("invalid active volume")
		}
		vol := s.guest.Volumes()[s.active]
		target := s.cwd
		if strings.TrimSpace(p) != "" {
			target = s.resolve(p)
		}
		entries, err := vol.ListDir(target)
		if err != nil {
			return nil, err
		}
		out := make([]EntryInfo, 0, len(entries))
		for _, e := range entries {
			out = append(out, EntryInfo{
				Name:      e.Name,
				Path:      e.Path,
				IsDir:     e.IsDir,
				SizeBytes: e.Size,
				SizeHuman: humanSize(e.Size),
			})
		}
		return out, nil
	}

	target := s.cwd
	if strings.TrimSpace(p) != "" {
		target = s.resolve(p)
	}

	item, err := s.v.Get(target, nil)
	if err != nil {
		return nil, err
	}

	if !item.IsDir() {
		size, _ := item.Size()
		return []EntryInfo{{
			Name:      item.Name,
			Path:      target,
			IsDir:     false,
			SizeBytes: size,
			SizeHuman: humanSize(size),
		}}, nil
	}

	entries, err := item.IterDir()
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	out := make([]EntryInfo, 0, len(entries))
	for _, e := range entries {
		size := uint64(0)
		if !e.IsDir() {
			size, _ = e.Size()
		}
		out = append(out, EntryInfo{
			Name:      e.Name,
			Path:      joinPath(target, e.Name),
			IsDir:     e.IsDir(),
			SizeBytes: size,
			SizeHuman: humanSize(size),
		})
	}

	return out, nil
}

func (s *Shell) FindMatches(name, start string) ([]string, error) {
	if s.guest != nil {
		if s.active < 0 || s.active >= len(s.guest.Volumes()) {
			return nil, fmt.Errorf("invalid active volume")
		}
		vol := s.guest.Volumes()[s.active]
		root := s.cwd
		if strings.TrimSpace(start) != "" {
			root = s.resolve(start)
		}
		needle := strings.ToLower(name)

		var out []string
		var walk func(string) error
		walk = func(cur string) error {
			entries, err := vol.ListDir(cur)
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.IsDir {
					if err := walk(e.Path); err != nil {
						continue
					}
					continue
				}
				if strings.Contains(strings.ToLower(e.Name), needle) {
					out = append(out, e.Path)
				}
			}
			return nil
		}
		if err := walk(root); err != nil {
			return nil, err
		}
		return out, nil
	}

	root := s.cwd
	if strings.TrimSpace(start) != "" {
		root = s.resolve(start)
	}
	needle := strings.ToLower(name)

	out := make([]string, 0, 16)
	err := s.walk(root, func(p string, item *vbk.DirItem) error {
		if item.IsDir() {
			return nil
		}
		if strings.Contains(strings.ToLower(item.Name), needle) {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Shell) Extract(src, dst string) (GetResult, error) {
	return s.ExtractWithOptions(src, dst, ExtractOptions{})
}

func joinPath(base, name string) string {
	if base == "/" {
		return "/" + name
	}
	if strings.HasSuffix(base, "/") {
		return base + name
	}
	return base + "/" + name
}
