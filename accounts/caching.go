// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains cache(ing) interface and associated helper structs and functions.

package accounts

import (
	"time"
	"github.com/ethereumproject/go-ethereum/common"
	"fmt"
	"os"
	"strings"
)

// Minimum amount of time between cache reloads. This limit applies if the platform does
// not support change notifications. It also applies if the keystore directory does not
// exist yet, the code will attempt to create a watcher at most this often.
const minReloadInterval = 2 * time.Second

type accountsByFile []Account

func (s accountsByFile) Len() int           { return len(s) }
func (s accountsByFile) Less(i, j int) bool { return s[i].File < s[j].File }
func (s accountsByFile) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// AmbiguousAddrError is returned when attempting to unlock
// an address for which more than one file exists.
type AmbiguousAddrError struct {
	Addr    common.Address
	Matches []Account
}

func (err *AmbiguousAddrError) Error() string {
	files := ""
	for i, a := range err.Matches {
		files += a.File
		if i < len(err.Matches)-1 {
			files += ", "
		}
	}
	return fmt.Sprintf("multiple keys match address (%s)", files)
}

type caching interface {
	muLock()
	muUnlock()
	getKeydir() string
	getWatcher() *watcher
	getThrottle() *time.Timer

	maybeReload()
	reload()
	Syncfs2db(time.Time) []error

	hasAddress(address common.Address) bool
	accounts() []Account
	add(Account)
	delete(Account)
	find(Account) (Account, error)

	close()
}


func skipKeyFile(fi os.FileInfo) bool {
	// Skip editor backups and UNIX-style hidden files.
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	if strings.HasSuffix(fi.Name(), "accounts.db") {
		return true
	}
	// Skip misc special files, directories (yes, symlinks too).
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}