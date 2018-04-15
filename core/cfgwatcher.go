package core

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

type configWatchdog struct {
	path    string
	watcher *fsnotify.Watcher
}

// NewConfigWatchdog creates new file watcher for config
// directory
func NewConfigWatchdog(path string) (*configWatchdog, error) {
	cw := new(configWatchdog)
	cw.path = path
	watcher, err := fsnotify.NewWatcher()
	cw.watcher = watcher

	err = cw.watcher.Add(cw.path)

	return cw, err
}

// Start config directory watching
func (cw *configWatchdog) Start() {
	go func() {
		for {
			select {
			case event := <-cw.watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err := <-cw.watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()
}

// Stop config directory watching
func (cw *configWatchdog) Stop() error {
	return cw.watcher.Close()
}
