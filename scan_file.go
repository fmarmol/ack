package main

import (
	"bufio"
	"io/fs"
	"strings"
)

func ScanFile(fsys fs.FS, filepath string, pattern string) ([]Result, error) {
	file, err := fsys.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ret := []Result{}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, pattern) {
			lineNum++
			continue
		}
		ret = append(ret, Result{Path: filepath, LineNum: lineNum + 1, Line: line})
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return ret, nil

}
