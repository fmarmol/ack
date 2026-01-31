package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/fmarmol/permos"
	"github.com/spf13/cobra"
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

var cmd = &cobra.Command{
	Use:   "ack [pattern]",
	Short: "lookup [pattern] in files in and under all directory defined at [--root]]",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expect on argument")
		}
		pattern := args[0]

		ignoreDirs, err := cmd.Flags().GetStringSlice(IGNORE_DIR)
		if err != nil {
			return err
		}
		ignoreExts, err := cmd.Flags().GetStringSlice(IGNORE_EXT)
		if err != nil {
			return err
		}
		root := cmd.Flag("root").Value.String()
		results := make(chan Result)
		errors := make(chan Error)
		sem := make(chan struct{}, 10)
		var wg sync.WaitGroup

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if d == nil {
				return nil
			}
			if len(ignoreDirs) > 0 && d.IsDir() && slices.Contains(ignoreDirs, d.Name()) {
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
		return nil
	},
}

const (
	IGNORE_DIR = "ignore-dir"
	IGNORE_EXT = "ignore-ext"
	ROOT       = "root"
)

func main() {

	fs := cmd.Flags()

	fs.StringSliceP("ignore-dir", "d", nil, "directories to ignore")
	fs.StringSliceP("ignore-ext", "e", nil, "extensions to ignore")
	fs.StringP("root", "r", ".", "root folder")

	// pattern := flag.String("pattern", "", "pattern to look for")
	// ignoreDir := flag.Stri("ignore-dir", "", "ignore the specified directory")
	// ignoreExt := flag.String("ignore-ext", "", "ingore the specified extension")
	// root := flag.String("root", ".", "root folder")
	// TODO add case insensitive option
	// var pattern string
	// var root string = "."
	// switch len(os.Args) {
	// case 1:
	// 	log.Fatal("expect one substring to search")
	// case 2:
	// 	pattern = os.Args[1]
	// case 3:
	// 	pattern = os.Args[1]
	// 	root = os.Args[2]
	// default:
	// 	log.Fatal("too many arguments")
	// }
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}

}
