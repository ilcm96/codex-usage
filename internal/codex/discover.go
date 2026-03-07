package codex

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type SessionFileInfo struct {
	Path  string
	Size  int64
	Mtime int64
}

func DiscoverSessionFiles(sessionsDir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries; report later via returned error if desired.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan sessions dir %s: %w", sessionsDir, err)
	}

	sort.Strings(files)
	return files, nil
}

func DiscoverSessionFilesWithInfo(sessionsDir string) ([]SessionFileInfo, error) {
	var files []SessionFileInfo

	err := filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			// Skip unreadable files.
			return nil
		}

		files = append(files, SessionFileInfo{
			Path:  path,
			Size:  info.Size(),
			Mtime: info.ModTime().UnixNano(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan sessions dir %s: %w", sessionsDir, err)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
