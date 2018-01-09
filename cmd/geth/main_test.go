package main

import (
	"testing"
	"strings"
	"os"
)

// TestSetVersionIfFromSourceWithOverride checks the conditional in the init fn.
func TestSetVersionIfFromSource(t *testing.T) {
	expectedSetVersionPrefix := "source_v"

	// Ensure version was set by init fn (without linker flag), it should already not be "source"
	if !strings.Contains(strings.Join(os.Args, " "), "ldflag") && Version == "source" {
		t.Errorf("Version should be set to git revision by default, got: %s", Version)
	}

	// Reset, ensure conditional Version override; double check fn functionality.
	Version = "source"
	setVersionIfDefaulty()
	if Version == "source" || !strings.Contains(Version, expectedSetVersionPrefix) {
		t.Errorf("Build from source did not set version. Got: %v", Version)
	} else {
		// Log for visual clarity and confirmation
		t.Log("OK: source version=", Version)
	}

	if strings.Contains(Version, "\\n") || strings.Contains(Version, "\\r") {
		t.Errorf("Got unwanted newline")
	}

	customVersion := "custom_ldflags_version"
	Version = customVersion
	setVersionIfDefaulty()
	if Version != customVersion {
		t.Error("Build from source with -ldflags override for main.Version (nondefaulty)")
	}
}
