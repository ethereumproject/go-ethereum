package core

import (
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type configWatchdog struct {
	path      string
	watcher   *fsnotify.Watcher
	observers []func(*SufficientChainConfig)
}

// NewConfigWatchdog creates new file watcher for config
// directory
func NewConfigWatchdog() (*configWatchdog, error) {
	path := flaggedExternalChainConfigPath // pull the package-level variable
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// monitor entire directory - we assume, that files can be included only from this directory
	err = watcher.Add(filepath.Dir(path))

	return &configWatchdog{path, watcher, nil}, err
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
					newConfig, err := ReadExternalChainConfigFromFile(cw.path)
					if err != nil {
						log.Println("configuration error:", err)
						continue
					}
					for _, observer := range cw.observers {
						observer(newConfig)
					}
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

func (cw *configWatchdog) Subscribe(observer func(*SufficientChainConfig)) {
	cw.observers = append(cw.observers, observer)
}
