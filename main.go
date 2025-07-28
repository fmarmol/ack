package main

import (
	"bufio"
	"fmt"
	"github.com/fmarmol/permos"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Result struct {
	Path    string
	Line    string
	LineNum int
}

type Error struct {
	Path string
	Err  error
}

func main() {
	// TODO add case insensitive option
	var pattern string
	var root string = "."
	switch len(os.Args) {
	case 1:
		log.Fatal("expect one substring to search")
	case 2:
		pattern = os.Args[1]
	case 3:
		pattern = os.Args[1]
		root = os.Args[2]
	default:
		log.Fatal("too many arguments")
	}

	results := make(chan Result)
	errors := make(chan Error)
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if d == nil {
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
			fd, err := os.Open(path)
			if err != nil {
				errors <- Error{Path: path, Err: err}
			}
			defer fd.Close()
			scanner := bufio.NewScanner(fd)
			lineNum := 0
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.Contains(line, pattern) {
					lineNum++
					continue
				}
				results <- Result{Path: path, LineNum: lineNum + 1, Line: line}
				lineNum++
			}
			<-sem
		}()
		return nil
	})

	finish := make(chan struct{}, 1)
	go func() {
		fmt.Println("--------Results--------")
		for result := range results {
			fmt.Printf("%s:%d: %s\n", result.Path, result.LineNum, result.Line)
		}
		fmt.Println("--------Errors---------")
		for err := range errors {
			fmt.Printf("%v: %s\n", err.Path, err.Err.Error())
		}
		finish <- struct{}{}
	}()
	wg.Wait()
	close(errors)
	close(results)
	if err != nil {
		fmt.Println("Scan finished with err:", err)
	}
	<-finish

}
