package main

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fmarmol/permos"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestIgnoreDirs(t *testing.T) {
	fsys := afero.NewMemMapFs()
	err := fsys.MkdirAll("a/b", permos.FileMode(permos.UserRead))
	require.NoError(t, err)
	err = fsys.MkdirAll("a/c", permos.FileMode(permos.UserRead))

	file1, err := fsys.Create("a/b/file1.txt")
	require.NoError(t, err)

	_, err = file1.WriteString("toto")
	require.NoError(t, err)
	file1.Close()

	file2, err := fsys.Create("a/c/file2.txt")
	require.NoError(t, err)

	_, err = file2.WriteString("toto")
	require.NoError(t, err)
	file2.Close()

	results := make(chan Result)
	errors := make(chan Error)

	var wg sync.WaitGroup

	err = WalkDir(afero.NewIOFS(fsys), ".", "toto", results, errors, []string{"a/b"}, nil, &wg)
	require.NoError(t, err)

	done := make(chan struct{}, 1)
	go func() {
		for result := range results {
			fmt.Println("res:", result)
		}
		done <- struct{}{}
	}()
	go func() {
		for err := range errors {
			fmt.Println("res:", err)
		}
		done <- struct{}{}
	}()
	wg.Wait()
	close(results)
	close(errors)
	<-done

}
