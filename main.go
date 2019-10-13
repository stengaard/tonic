// tonic is a live reload tool for 12 factor apps
//
// tonic monitors a directory a reloads the daemon if any .go source
// files change.
package main

import (
	"io/ioutil"
	"strings"

	gin "github.com/codegangsta/gin/lib"
	"github.com/fsnotify/fsnotify"

	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	startTime = time.Now()
	logger    = log.New(os.Stdout, "[tonic] ", 0)
)

func usage() {
	header := `tonic is a live reload utility for Go apps.

Usage of tonic:
`
	out := flag.CommandLine.Output()
	fmt.Fprintln(out, header)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	var (
		path      = flag.String("path", ".", "Path to recursively watch files under")
		bin       = flag.String("bin", "tonic-bin", "Name of generated binary")
		buildArgs = flag.String("build-args", "", "Extra build ars to pass in")
	)
	flag.Parse()
	if err := run(*path, *bin, *buildArgs, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "tonic: %v\n", err)
		os.Exit(1)
	}
}

func run(path, bin, buildArgs string, args []string) error {

	// Avoid littering in the local dir, create a tempfile.
	dir, err := ioutil.TempDir("", "tonic-bin")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	builder := gin.NewBuilder(".", bin, false, dir, strings.Fields(buildArgs))
	runner := gin.NewRunner(filepath.Join(dir, bin), args...)
	runner.SetWriter(os.Stdout)

	err = build(builder, logger)
	if err == nil {
		runner.Run()
	}

	w, err := watch(path)
	if err != nil {
		return err
	}

	var t <-chan time.Time
	for {
		select {
		case ev := <-w.Events:
			if ofInterest(ev) {
				// debounce
				if t == nil {
					t = time.After(100 * time.Millisecond)
				}
			}
		case <-t:
			runner.Kill()
			err := build(builder, logger)
			if err == nil {
				runner.Run()
			}
			t = nil

		case err := <-w.Errors:
			return err
		}
	}
}

func build(builder gin.Builder, logger *log.Logger) error {
	err := builder.Build()
	if err != nil {
		logger.Println("Err - Build failed!")
		fmt.Println(builder.Errors())
	} else {
		logger.Println("Build successful")
	}
	time.Sleep(100 * time.Millisecond)
	return err
}

func ofInterest(ev fsnotify.Event) bool {
	if ev.Name == ".git" {
		return false
	}

	if filepath.Base(ev.Name)[0] == '.' {
		return false
	}

	if filepath.Ext(ev.Name) == ".go" {
		return true
	}
	return false
}

func watch(root string) (*fsnotify.Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return w, addAllDirs(w, root)

}

func addAllDirs(w *fsnotify.Watcher, root string) error {
	err := w.Add(root)
	if err != nil {
		return err
	}
	return filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		if i.IsDir() {
			return w.Add(p)
		}
		return nil
	})
}
