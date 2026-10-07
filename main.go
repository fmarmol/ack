package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

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
		// trim extra slash
		ignoreDirs = func() (ret []string) {
			for _, dir := range ignoreDirs {
				dirTrimed := dir
				dirTrimed = strings.TrimSuffix(dirTrimed, "\\")
				dirTrimed = strings.TrimSuffix(dirTrimed, "/")
				ret = append(ret, dirTrimed)
			}
			return
		}()
		ignoreExts, err := cmd.Flags().GetStringSlice(IGNORE_EXT)
		if err != nil {
			return err
		}
		root := cmd.Flag("root").Value.String()
		results := make(chan Result)
		errors := make(chan Error)
		var wg sync.WaitGroup

		err = WalkDir(os.DirFS(root), root, pattern, results, errors, ignoreDirs, ignoreExts, &wg)
		if err != nil {
			return err
		}

		var wgRead sync.WaitGroup

		wgRead.Add(2)
		go func() {
			defer wgRead.Done()
			fmt.Println("--------Results--------")
			for result := range results {
				fmt.Printf("%s:%d: %s\n", result.Path, result.LineNum, result.Line)
			}
		}()
		go func() {
			defer wgRead.Done()
			fmt.Println("--------Errors---------")
			for err := range errors {
				fmt.Printf("%v: %s\n", err.Path, err.Err.Error())
			}

		}()
		wg.Wait()
		close(errors)
		close(results)
		wgRead.Wait()
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
