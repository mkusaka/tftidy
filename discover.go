package tftidy

import (
	"github.com/boyter/gocodewalker"
	"path/filepath"
	"sort"
)

func discoverFiles(dir string) ([]string, error) {
	fileCh := make(chan *gocodewalker.File, 256)
	walker := gocodewalker.NewFileWalker(dir, fileCh)
	walker.AllowListExtensions = append(walker.AllowListExtensions, "tf")
	walker.ExcludeDirectory = append(walker.ExcludeDirectory, ".terraform", ".terragrunt-cache")

	errCh := make(chan error, 1)
	go func() {
		errCh <- walker.Start()
	}()

	files := make([]string, 0, 256)
	for file := range fileCh {
		files = append(files, filepath.Clean(file.Location))
	}

	if err := <-errCh; err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}
