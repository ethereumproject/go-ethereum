package main

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		value    string
		err      bool
		expected time.Duration
	}{
		// valid ones
		{"123", false, 123 * time.Second},
		{"10h", false, 10 * time.Hour},
		{"3d", false, 3 * 24 * time.Hour},
		{"4w", false, 4 * 7 * 24 * time.Hour},
		// same with whitespace
		{"  123", false, 123 * time.Second},
		{"10h	", false, 10 * time.Hour},
		{"3d  ", false, 3 * 24 * time.Hour},
		{"  4w   ", false, 4 * 7 * 24 * time.Hour},
		{"  4 w   ", false, 4 * 7 * 24 * time.Hour},
		// upper case
		{"10H", false, 10 * time.Hour},
		{"3D", false, 3 * 24 * time.Hour},
		{"4W", false, 4 * 7 * 24 * time.Hour},
		// invalid cases
		{"1y", true, 0},
		{"one two", true, 0},
		{"d", true, 0},
		{"1week", true, 0},
		{"10www", true, 0},
		{"", true, 0},
		{"r2d2", true, 0},
		{"2d2d", true, 0},
	}

	for _, test := range cases {
		t.Run(test.value, func(t *testing.T) {
			actual, err := parseDuration(test.value)
			if test.err != (err != nil) {
				t.Errorf("expected error: %v, found error: %v", test.err, err)
			}
			if test.expected != actual {
				t.Errorf("expected: %v, actual: %v", test.expected, actual)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	cases := []struct {
		value    string
		err      bool
		expected uint64
	}{
		// valid ones
		{"123", false, 123},
		{"10k", false, 10 * 1024},
		{"3m", false, 3 * 1024 * 1024},
		{"4g", false, 4 * 1024 * 1024 * 1024},
		// same with whitespace
		{"123  ", false, 123},
		{"	10k", false, 10 * 1024},
		{"  3m", false, 3 * 1024 * 1024},
		{" 4g ", false, 4 * 1024 * 1024 * 1024},
		{" 4	g ", false, 4 * 1024 * 1024 * 1024},
		// upper case
		{"10K", false, 10 * 1024},
		{"3M", false, 3 * 1024 * 1024},
		{"4G", false, 4 * 1024 * 1024 * 1024},
		// invalid cases
		{"1t", true, 0},
		{"one two", true, 0},
		{"d", true, 0},
		{"2gigs", true, 0},
		{"321petabytes", true, 0},
		{"", true, 0},
		{"r2d2", true, 0},
		{"2d2d", true, 0},
	}

	for _, test := range cases {
		t.Run(test.value, func(t *testing.T) {
			actual, err := parseSize(test.value)
			if test.err != (err != nil) {
				t.Errorf("expected error: %v, found error: %v", test.err, err)
			}
			if test.expected != actual {
				t.Errorf("expected: %v, actual: %v", test.expected, actual)
			}
		})
	}
}
