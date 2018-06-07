package main

import (
	"fmt"
	"github.com/ethereumproject/benchmark/rtprof"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/metrics"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
	"path/filepath"
	"time"
)

func appBeforeContext(ctx *cli.Context) error {

	if ctx.NArg() > 0 && ctx.Args()[0] == cmdDumpAppConfig.Name {
		if err := dumpAppConfig(ctx); err != nil {
			log.Fatalln(err)
		}
		os.Exit(0)
	}

	// It's a patch.
	// Don't know why urfave/cli isn't catching the unknown command on its own.
	if ctx.Args().Present() {
		commandExists := false
		for _, cmd := range ctx.App.Commands {
			if cmd.HasName(ctx.Args().First()) {
				commandExists = true
			}
		}
		if !commandExists {
			if e := cli.ShowCommandHelp(ctx, ctx.Args().First()); e != nil {
				return e
			}
		}
	}

	// Read from flagged --app-config <config.toml> file,
	// or if file exists at <datadir>/<chain>/appconfig.toml then use that.
	if ctx.IsSet(AppConfigFileFlag.Name) {
		p := filepath.Clean(ctx.GlobalString(AppConfigFileFlag.Name))
		mustReadAppConfigFromFile(ctx, p)
	} else {
		p := filepath.Join(MustMakeChainDataDir(ctx), "appconfig.toml")
		if _, err := os.Stat(p); err == nil {
			mustReadAppConfigFromFile(ctx, p)
		}
	}

	// Check for --exec set without console OR attach
	if ctx.IsSet(ExecFlag.Name) {
		// If no command is used, OR command is not one of the valid commands attach/console
		if cmdName := ctx.Args().First(); cmdName == "" || (cmdName != "console" && cmdName != "attach") {
			log.Printf("Error: --%v flag requires use of 'attach' OR 'console' command, command was: '%v'", ExecFlag.Name, cmdName)
			cli.ShowCommandHelp(ctx, consoleCommand.Name)
			cli.ShowCommandHelp(ctx, attachCommand.Name)
			os.Exit(1)
		}
	}

	if ctx.IsSet(SputnikVMFlag.Name) {
		if core.SputnikVMExists {
			core.UseSputnikVM = true
		} else {
			log.Fatal("This version of geth wasn't built to include SputnikVM. To build with SputnikVM, use -tags=sputnikvm following the go build command.")
		}
	}

	// Check for migrations and handle if conditionals are met.
	if err := handleIfDataDirSchemaMigrations(ctx); err != nil {
		return err
	}

	if err := setupLogRotation(ctx); err != nil {
		return err
	}

	// Handle parsing and applying log verbosity, severities, and default configurations from context.
	if err := setupLogging(ctx); err != nil {
		return err
	}

	// Handle parsing and applying log rotation configs from context.
	if err := setupLogRotation(ctx); err != nil {
		return err
	}

	if s := ctx.String("metrics"); s != "" {
		go metrics.CollectToFile(s)
	}

	// This should be the only place where reporting is enabled
	// because it is not intended to run while testing.
	// In addition to this check, bad block reports are sent only
	// for chains with the main network genesis block and network id 1.
	eth.EnableBadBlockReporting = true

	// (whilei): I use `log` instead of `glog` because git diff tells me:
	// > The output of this command is supposed to be machine-readable.
	gasLimit := ctx.GlobalString(aliasableName(TargetGasLimitFlag.Name, ctx))
	if _, ok := core.TargetGasLimit.SetString(gasLimit, 0); !ok {
		return fmt.Errorf("malformed %s flag value %q", aliasableName(TargetGasLimitFlag.Name, ctx), gasLimit)
	}

	// Set morden chain by default for dev mode.
	if ctx.GlobalBool(aliasableName(DevModeFlag.Name, ctx)) {
		if !ctx.GlobalIsSet(aliasableName(ChainIdentityFlag.Name, ctx)) {
			if e := ctx.Set(aliasableName(ChainIdentityFlag.Name, ctx), "morden"); e != nil {
				return fmt.Errorf("failed to set chain value: %v", e)
			}
		}
	}

	if port := ctx.GlobalInt(PprofFlag.Name); port != 0 {
		interval := 5 * time.Second
		if i := ctx.GlobalInt(PprofIntervalFlag.Name); i > 0 {
			interval = time.Duration(i) * time.Second
		}
		rtppf.Start(interval, port)
	}

	return nil
}
