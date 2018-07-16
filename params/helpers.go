// +build !windows

package params

import (
	"fmt"
	"io"

	"github.com/ethereumproject/go-ethereum/params/assets"
)

func assetsOpen(path string) (io.ReadCloser, error) {
	file, err := assets.DEFAULTS.Open(path)
	if err != nil {
		err = fmt.Errorf("Error opening '%s' default JSON: %v", path, err)
	}
	return file, err
}
