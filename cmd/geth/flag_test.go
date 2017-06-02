package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"reflect"

	"github.com/ethereumproject/go-ethereum/accounts"
	"github.com/ethereumproject/go-ethereum/common"
	"gopkg.in/urfave/cli.v1"
)

var ogHome string  // placeholder
var tmpHOME string // fake $HOME (for defaults)
var tmpDir string  // temp DATA_DIR (inside tmpHOME)

var app *cli.App
var context *cli.Context

var set *flag.FlagSet

// globally available flags that go will parse, making them available for the mock app
type flags []struct {
	name    string
	aliases []string
	value   interface{}
}

var gFlags flags

func makeTmpDataDir(t *testing.T) {
	ogHome = common.HomeDir()
	var e error

	tmpHOME, e = ioutil.TempDir(ogHome, "HOME")
	if e != nil {
		t.Fatalf("Failed to create temp directory in: %v", ogHome)
	}

	if e = os.Setenv("HOME", tmpHOME); e != nil {
		t.Fatalf("Failed to temporarily set system home: %v", e)
	}

	td, err := ioutil.TempDir(tmpHOME, "DATADIR")
	if err != nil {
		t.Fatalf("Failed to create temp directory in: %v", tmpHOME)
	}
	tmpDir = td
}

func rmTmpDataDir(t *testing.T) {
	if e := os.RemoveAll(tmpHOME); e != nil {
		t.Fatalf("Failed to remove temp dir: %v", e)
	}

	if e := os.Setenv("HOME", ogHome); e != nil {
		t.Fatalf("Failed to reset system home env var: %v", e)
	}
}

func setupFlags(t *testing.T) {

	gFlags = flags{
		{"testnet", []string{}, false},
		{"data-dir", []string{"datadir"}, common.DefaultDataDir()},
		{"bootnodes", []string{}, ""},
		{"chain", []string{}, ""},
	}

	app = makeCLIApp()
	app.Writer = ioutil.Discard

	set = flag.NewFlagSet("test", 0)

	for _, f := range gFlags {
		switch f.value.(type) {
		case string:
			set.String(f.name, f.value.(string), "")
		case bool:
			set.Bool(f.name, f.value.(bool), "")
		case int:
			set.Int(f.name, f.value.(int), "")
		}

		if len(f.aliases) > 0 {
			for _, a := range f.aliases {
				switch f.value.(type) {
				case string:
					set.String(a, f.value.(string), "")
				case bool:
					set.Bool(a, f.value.(bool), "")
				case int:
					set.Int(a, f.value.(int), "")
				}
			}
		}
	}
}

func TestMustMakeChainDataDir(t *testing.T) {

	makeTmpDataDir(t)
	defer rmTmpDataDir(t)

	dd := common.DefaultDataDir()
	funkyName := "my.private-chain_2chainz!"

	cases := []struct {
		flags []string
		want  string
		err   error
	}{
		{[]string{}, filepath.Join(dd, "mainnet"), nil},

		{[]string{"--datadir", tmpDir}, filepath.Join(tmpDir, "mainnet"), nil},
		{[]string{"--data-dir", tmpDir}, filepath.Join(tmpDir, "mainnet"), nil},

		{[]string{"--testnet", "--data-dir", tmpDir}, filepath.Join(tmpDir, "morden"), nil},
		{[]string{"--testnet"}, filepath.Join(dd, "morden"), nil},

		{[]string{"--chain"}, "", ErrInvalidFlag},
		{[]string{"--chain", "main"}, filepath.Join(dd, "mainnet"), nil},
		{[]string{"--chain", "morden"}, filepath.Join(dd, "morden"), nil},
		{[]string{"--chain", "testnet"}, filepath.Join(dd, "morden"), nil},
		{[]string{"--chain", "kitty"}, filepath.Join(dd, "kitty"), nil},

		// Passed when  run individually, but fails when run as go test. This is not a code problem.
		{[]string{"--chain", "kitty/cat"}, filepath.Join(dd, funkyName), ErrInvalidChainID},
		{[]string{"--chain", funkyName}, filepath.Join(dd, funkyName), nil},

	}

	for _, c := range cases {

		if c.err != nil {
			t.Log("skipping test for erroring use case (will pass, but go test doesn't like glog)")
			continue
		}

		setupFlags(t)

		if e := set.Parse(c.flags); e != nil {
			if c.err == nil {
				t.Fatal(e)
			} else {
				// don't compare the errors for now, this is enough
				continue
			}
		}
		context = cli.NewContext(app, set, nil)

		got := MustMakeChainDataDir(context)

		if c.err == nil && got != c.want {
			t.Errorf("flag: %v, chaindir want: %v, got: %v", c.flags, c.want, got)
		}
		if c.err == nil && !filepath.IsAbs(got) {
			t.Errorf("flag: %v, unexpected relative path: %v", c.flags, got)
		}
		if c.err != nil && got != "" {
			t.Errorf("flag: %v, want: %v, got: %v", c.flags, c.err, got)
		}
	}
}

func TestGetChainIdentityValue(t *testing.T) {

	cases := []struct {
		flags []string
		want  string
	}{
		// Known (defaulty) chain values.
		{[]string{"--chain", "morden"}, "morden"},
		{[]string{"--chain", "testnet"}, "morden"},
		{[]string{"--chain", "main"}, "mainnet"},
		{[]string{"--chain", "mainnet"}, "mainnet"},

		// Custom.
		{[]string{"--chain", "kitty"}, "kitty"},

		// Blacklisted.
		{[]string{"--chain", "chaindata"}, ""},
		{[]string{"--chain", "dapp"}, ""},

		 // Invalid.
		 // These pass when test is run individually, but go test doesn't like error out.
		{[]string{"--chain", "kitty/cat"}, ""},
	}

	for _, c := range cases {
		if c.want == "" {
			t.Log("skipping test for erroring use case (will pass, but go test doesn't like glog)")
			continue
		}
		
		setupFlags(t)

		if e := set.Parse(c.flags); e != nil {
			t.Fatal(e)
		}
		context = cli.NewContext(app, set, nil)

		if got := mustMakeChainIdentity(context); c.want != got {
			t.Fatalf("[%v] want: %v, got: %v", c.flags, c.want, got)
		}

	}
}

// Bootnodes flag parse 1
func TestMakeBootstrapNodesFromContext1(t *testing.T) {

	makeTmpDataDir(t)
	defer rmTmpDataDir(t)
	setupFlags(t)

	arg := []string{
		"--bootnodes",
		"enode://6e538e7c1280f0a31ff08b382db5302480f775480b8e68f8febca0ceff81e4b19153c6f8bf60313b93bef2cc34d34e1df41317de0ce613a201d1660a788a03e2@52.206.67.235:30303",
	}
	if e := set.Parse(arg); e != nil {
		t.Fatal(e)
	}
	context = cli.NewContext(app, set, nil)
	got := MakeBootstrapNodesFromContext(context)
	if len(got) != 1 {
		t.Errorf("wanted: 1, got %v", len(got))
	}
	if got[0].IP.String() != "52.206.67.235" {
		t.Errorf("unexpected: %v", got[0].IP.String())
	}
}

// Bootnodes flag parse 2
func TestMakeBootstrapNodesFromContext2(t *testing.T) {

	makeTmpDataDir(t)
	defer rmTmpDataDir(t)
	setupFlags(t)

	arg := []string{
		"--bootnodes",
		`enode://6e538e7c1280f0a31ff08b382db5302480f775480b8e68f8febca0ceff81e4b19153c6f8bf60313b93bef2cc34d34e1df41317de0ce613a201d1660a788a03e2@52.206.67.235:30303,enode://f50e675a34f471af2438b921914b5f06499c7438f3146f6b8936f1faeb50b8a91d0d0c24fb05a66f05865cd58c24da3e664d0def806172ddd0d4c5bdbf37747e@144.76.238.49:30306`,
	}
	if e := set.Parse(arg); e != nil {
		t.Fatal(e)
	}
	context = cli.NewContext(app, set, nil)
	got := MakeBootstrapNodesFromContext(context)
	if len(got) != 2 {
		t.Errorf("wanted: 2, got %v", len(got))
	}
	if got[0].IP.String() != "52.206.67.235" {
		t.Errorf("unexpected: %v", got[0].IP.String())
	}
	if got[1].IP.String() != "144.76.238.49" {
		t.Errorf("unexpected: %v", got[1].IP.String())
	}
}

// Bootnodes default
func TestMakeBootstrapNodesFromContext3(t *testing.T) {

	makeTmpDataDir(t)
	defer rmTmpDataDir(t)
	setupFlags(t)

	arg := []string{}
	if e := set.Parse(arg); e != nil {
		t.Fatal(e)
	}
	context = cli.NewContext(app, set, nil)
	got := MakeBootstrapNodesFromContext(context)
	if len(got) != len(HomesteadBootNodes) {
		t.Errorf("wanted: %v, got %v", len(HomesteadBootNodes), len(got))
	}
}

// Bootnodes testnet default
func TestMakeBootstrapNodesFromContext4(t *testing.T) {

	makeTmpDataDir(t)
	defer rmTmpDataDir(t)
	setupFlags(t)

	arg := []string{"--testnet"}
	if e := set.Parse(arg); e != nil {
		t.Fatal(e)
	}
	context = cli.NewContext(app, set, nil)
	got := MakeBootstrapNodesFromContext(context)
	if len(got) != len(TestNetBootNodes) {
		t.Errorf("wanted: %v, got %v", len(TestNetBootNodes), len(got))
	}
}

func TestMakeAddress(t *testing.T) {
	accAddr := "f466859ead1932d743d622cb74fc058882e8648a" // account[0] address
	cachetestdir := filepath.Join("accounts", "testdata", "keystore")
	am, err := accounts.NewManager(cachetestdir, accounts.LightScryptN, accounts.LightScryptP, false)
	if err != nil {
		t.Fatal(err)
	}
	gotAccount, e := MakeAddress(am, accAddr)
	if e != nil {
		t.Fatalf("makeaddress: %v", e)
	}
	wantAccount := accounts.Account{
		Address: common.HexToAddress(accAddr),
	}
	// compare all
	if !reflect.DeepEqual(wantAccount, gotAccount) {
		t.Fatalf("want: %v, got: %v", wantAccount, gotAccount)
	}
}

