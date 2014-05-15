// tonic is a live reload tool for 12 factor apps
//
// tonic monitors a directory a reloads the daemon if any .go source
// files change.
package main

import (
	"code.google.com/p/go.exp/fsnotify"
	"github.com/codegangsta/cli"
	gin "github.com/codegangsta/gin/lib"

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

func main() {
	app := cli.NewApp()
	app.Name = "tonic"
	app.Usage = "A live reload utility for Go apps"
	app.Action = RunCommand
	app.Flags = []cli.Flag{
		cli.StringFlag{"path,t", ".", "Path to watch files from"},
		cli.StringFlag{"bin,b", "tonic-bin", "Name of generated binary file"},
	}

	app.Commands = []cli.Command{
		{
			Name:      "run",
			ShortName: "r",
			Usage:     "Build and run the given command",
			Action:    RunCommand,
		},
	}

	app.Run(os.Args)
}

func RunCommand(c *cli.Context) {

	wd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}

	builder := gin.NewBuilder(".", c.GlobalString("bin"))
	runner := NewRunner(filepath.Join(wd, builder.Binary()), c.Args())
	runner.SetWriter(os.Stdout)

	err = build(builder, logger)
	if err == nil {
		runner.Run()
	}

	w, err := watch(c.GlobalString("path"))
	if err != nil {
		logger.Fatal(err)
	}

	var t <-chan time.Time
	for {
		select {
		case ev := <-w.Event:
			if ofInterest(ev) {
				//logger.Println(ev)
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

		case err := <-w.Error:
			logger.Fatal(err)
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

func ofInterest(ev *fsnotify.FileEvent) bool {
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
	err := w.Watch(root)
	if err != nil {
		return err
	}
	return filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		if i.IsDir() {
			return w.Watch(p)
		}
		return nil
	})
}
