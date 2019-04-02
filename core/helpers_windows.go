package core

import (
	"fmt"
	"io"
	"strings"

	"github.com/eth-classic/go-ethereum/core/assets"
)

func assetsOpen(path string) (io.ReadCloser, error) {
	// this is dirty hack, but /such/path is considered as NOT absolute path on windows,
	// and because of that we process the path as relative. But we know, that asset paths
	// are always absolute. Moreover, we need to fish slashes ("broken" by filepath.Join).
	path = "/" + strings.Replace(path, "\\", "/", -1)

	file, err := assets.DEFAULTS.Open(path)
	if err != nil {
		err = fmt.Errorf("Error opening '%s' default JSON: %v", path, err)
	}
	return file, err
}
