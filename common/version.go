package common

import (
	"path/filepath"
	"os/exec"
	"regexp"
	"runtime"
)

func sourceBuildVersion() string {
	// Get the path of this file
	_, f, _, ok := runtime.Caller(1)
	if ok {
		d := filepath.Dir(f) // .../cmd/geth
		// Derive git project dir
		d = filepath.Join(d, "..", "..", ".git")
		// Ignore error
		if o, err := exec.Command("git", "--git-dir", d, "describe", "--tags").Output(); err == nil {
			// Remove newline
			re := regexp.MustCompile(`\r?\n`) // Handle both Windows carriage returns and *nix newlines
			return re.ReplaceAllString(string(o), "")
		}
	}
	return ""
}

func SourceBuildVersionFormatted() string {
	if v := sourceBuildVersion(); v != "" {
		return "source_" + v
	} else {
		return "source_unknown"
	}
}