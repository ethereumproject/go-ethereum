package main

import (
	"testing"
	"io/ioutil"
	"flag"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"github.com/ethereumproject/go-ethereum/common"
)

var app *cli.App
var context *cli.Context
var tmpDir string

var set *flag.FlagSet
//var datadir string

func makeTmpDataDir(t *testing.T) {
	home := common.HomeDir()

	td, err := ioutil.TempDir(home, "tmp")
	if err != nil {
		t.Fatalf("Failed to create temp directory in: %v", home)
	}
	tmpDir = td
}

func rmTmpDataDir(t *testing.T) {
	if e := os.RemoveAll(tmpDir); e != nil {
		t.Fatalf("Failed to remove temp dir: %v", e)
	}
}

func setupFlags(t *testing.T) {
	app = makeCLIApp()
	app.Writer = ioutil.Discard

	set = flag.NewFlagSet("test", 0)

	set.String("datadir", "", "")
}

func TestMustMakeChainDataDir(t *testing.T) {

	makeTmpDataDir(t)
	setupFlags(t)

	args := []string{"--datadir", tmpDir}
	if e := set.Parse(args); e != nil {
		t.Fatal(e)
	}

	context = cli.NewContext(app, set, nil)

	got := MustMakeChainDataDir(context)
	expected := filepath.Join(tmpDir, "mainnet")
	if got != expected {
		t.Errorf("chaindir want: %v, got: %v", expected, got)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("unexpected relative path: %v", got)
	}

	rmTmpDataDir(t)
}

//func TestMustMakeChainConfig(t *testing.T) {
//
//	cases := []struct{
//		args []string
//		expectedResult interface{}
//		expectedErr error
//	}{
//		{[]string{"geth", "data-dir", tmpDir}, nil, nil},
//	}
//
//	for _, c := range cases {
//		makeTmpDataDir(c.args, t)
//		cmd := cli.Command{
//			Action: version,
//				Name:   "version",
//				Usage:  "print ethereum version numbers",
//				Description: `
//	The output of this command is supposed to be machine-readable.
//	`,
//		}
//		err := cmd.Run(context)
//		if err != c.expectedErr {
//			t.Errorf("error: %v", err)
//		}
//		rmTmpDataDir(t)
//	}
//
//}

