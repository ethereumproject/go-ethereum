package core

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

type configWatchdog struct {
	path    string
	watcher *fsnotify.Watcher
	notify  chan struct{}
}

// NewConfigWatchdog creates new file watcher for config
// directory
func NewConfigWatchdog(path string) (*configWatchdog, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(path)

	return &configWatchdog{path, watcher, make(chan struct{})}, err
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
					//cw.notify <- event
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

// GetNotifyChan returns channel notifications about config changes
func (cw *configWatchdog) GetNotifyChan() chan struct{} {
	return cw.notify
}
