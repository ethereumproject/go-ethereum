// Copyright 2015 The go-ethereum Authors
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

package metrics

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/ethereumproject/go-ethereum/logger/glog"
)

func readDiskStats(stats *diskStats) {
	file := fmt.Sprintf("/proc/%d/io", os.Getpid())
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		glog.Errorf("%s: %s", file, err)
		return
	}

	for _, line := range strings.Split(string(bytes), "\n") {
		i := strings.Index(line, ": ")
		if i < 0 {
			continue
		}

		var p *int64
		switch line[:i] {
		case "syscr":
			p = &stats.ReadCount
		case "syscw":
			p = &stats.WriteCount
		case "rchar":
			p = &stats.ReadBytes
		case "wchar":
			p = &stats.WriteBytes
		default:
			continue
		}

		*p, err = strconv.ParseInt(line[i+2:], 10, 64)
		if err != nil {
			glog.Errorf("%s: line %q: %s", file, line, err)
		}
	}
}
