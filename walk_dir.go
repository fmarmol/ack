package main

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/fmarmol/permos"
)

func WalkDir(fsys fs.FS, root string, pattern string, results chan Result, errors chan Error, ignoreDirs, ignoreExts []string, wg *sync.WaitGroup) error {
	sem := make(chan struct{}, 10)
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if d == nil {
			return nil
		}
		if len(ignoreDirs) > 0 && d.IsDir() && slices.Contains(ignoreDirs, path) {
			return filepath.SkipDir
		}

		if len(ignoreExts) > 0 && slices.ContainsFunc(ignoreExts, func(e string) bool {
			return strings.HasSuffix(d.Name(), e)
		}) {
			return nil
		}

		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.FileMode(permos.UserExec) == fs.FileMode(permos.UserExec) {
			return nil
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			resultsx, err := ScanFile(fsys, path, pattern)
			if err != nil {
				errors <- Error{Path: path, Err: err}
			} else {
				for _, res := range resultsx {
					results <- res
				}
			}
			<-sem
		}()
		return nil
	})
	return err
}
