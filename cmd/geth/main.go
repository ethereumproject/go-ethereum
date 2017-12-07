// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// geth is the official command-line client for Ethereum.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/console"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/metrics"
)

// Version is the application revision identifier. It can be set with the linker
// as in: go build -ldflags "-X main.Version="`git describe --tags`
var Version = "source"

func makeCLIApp() (app *cli.App) {
	app = cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = Version
	app.Usage = "the go-ethereum command line interface"
	app.Action = geth
	app.HideVersion = true // we have a command to print the version

	app.Commands = []cli.Command{
		importCommand,
		exportCommand,
		dumpChainConfigCommand,
		upgradedbCommand,
		removedbCommand,
		dumpCommand,
		rollbackCommand,
		recoverCommand,
		resetCommand,
		monitorCommand,
		accountCommand,
		walletCommand,
		consoleCommand,
		attachCommand,
		javascriptCommand,
		statusCommand,
		apiCommand,
		{
			Action:  makedag,
			Name:    "make-dag",
			Aliases: []string{"makedag"},
			Usage:   "Generate ethash dag (for testing)",
			Description: `
	The makedag command generates an ethash DAG in /tmp/dag.

	This command exists to support the system testing project.
	Regular users do not need to execute it.
			`,
		},
		{
			Action:  gpuinfo,
			Name:    "gpu-info",
			Aliases: []string{"gpuinfo"},
			Usage:   "GPU info",
			Description: `
	Prints OpenCL device info for all found GPUs.
			`,
		},
		{
			Action:  gpubench,
			Name:    "gpu-bench",
			Aliases: []string{"gpubench"},
			Usage:   "Benchmark GPU",
			Description: `
	Runs quick benchmark on first GPU found.
			`,
		},
		{
			Action: version,
			Name:   "version",
			Usage:  "Print ethereum version numbers",
			Description: `
	The output of this command is supposed to be machine-readable.
			`,
		},
		{
			Action: makeMLogDocumentation,
			Name:   "mdoc",
			Usage:  "Generate mlog documentation",
			Description: `
	Auto-generates documentation for all available mlog lines.
	Use -md switch to toggle markdown output (eg. for wiki).
	Arguments may be used to specify exclusive candidate components;
	so 'geth mdoc -md discover' will generate markdown documentation only
	for the 'discover' component.
			`,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "md",
					Usage: "Toggle markdown formatting",
				},
			},
		},
	}

	app.Flags = []cli.Flag{
		NodeNameFlag,
		UnlockedAccountFlag,
		PasswordFileFlag,
		AccountsIndexFlag,
		BootnodesFlag,
		DataDirFlag,
		DocRootFlag,
		KeyStoreDirFlag,
		ChainIdentityFlag,
		BlockchainVersionFlag,
		FastSyncFlag,
		CacheFlag,
		LightKDFFlag,
		JSpathFlag,
		ListenPortFlag,
		MaxPeersFlag,
		MaxPendingPeersFlag,
		EtherbaseFlag,
		GasPriceFlag,
		MinerThreadsFlag,
		MiningEnabledFlag,
		MiningGPUFlag,
		AutoDAGFlag,
		TargetGasLimitFlag,
		NATFlag,
		NatspecEnabledFlag,
		NoDiscoverFlag,
		NodeKeyFileFlag,
		NodeKeyHexFlag,
		RPCEnabledFlag,
		RPCListenAddrFlag,
		RPCPortFlag,
		RPCApiFlag,
		WSEnabledFlag,
		WSListenAddrFlag,
		WSPortFlag,
		WSApiFlag,
		WSAllowedOriginsFlag,
		IPCDisabledFlag,
		IPCApiFlag,
		IPCPathFlag,
		ExecFlag,
		PreloadJSFlag,
		WhisperEnabledFlag,
		DevModeFlag,
		TestNetFlag,
		NetworkIdFlag,
		RPCCORSDomainFlag,
		NeckbeardFlag,
		VerbosityFlag,
		DisplayFlag,
		VModuleFlag,
		LogDirFlag,
		LogStatusFlag,
		MLogFlag,
		MLogDirFlag,
		MLogComponentsFlag,
		BacktraceAtFlag,
		MetricsFlag,
		FakePoWFlag,
		SolcPathFlag,
		GpoMinGasPriceFlag,
		GpoMaxGasPriceFlag,
		GpoFullBlockRatioFlag,
		GpobaseStepDownFlag,
		GpobaseStepUpFlag,
		GpobaseCorrectionFactorFlag,
		ExtraDataFlag,
		Unused1,
	}

	app.Before = func(ctx *cli.Context) error {

		// It's a patch.
		// Don't know why urfave/cli isn't catching the unknown command on its own.
		if ctx.Args().Present() {
			commandExists := false
			for _, cmd := range app.Commands {
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

		runtime.GOMAXPROCS(runtime.NumCPU())

		glog.CopyStandardLogTo("INFO")

		// Data migrations if should.
		glog.SetToStderr(true)
		// Turn verbosity down for migration check. If migration happens, it will print to Warn.
		// Otherwise logs are just debuggers.
		glog.SetV(3)
		if shouldAttemptDirMigration(ctx) {
			// Rename existing default datadir <home>/<Ethereum>/ to <home>/<EthereumClassic>.
			// Only do this if --datadir flag is not specified AND <home>/<EthereumClassic> does NOT already exist (only migrate once and only for defaulty).
			// If it finds an 'Ethereum' directory, it will check if it contains default ETC or ETHF chain data.
			// If it contains ETC data, it will rename the dir. If ETHF data, if will do nothing.
			if migrationError := migrateExistingDirToClassicNamingScheme(ctx); migrationError != nil {
				glog.Fatalf("%v: failed to migrate existing Classic database: %v", ErrDirectoryStructure, migrationError)
			}

			// Move existing mainnet data to pertinent chain-named subdir scheme (ie ethereum-classic/mainnet).
			// This should only happen if the given (newly defined in this protocol) subdir doesn't exist,
			// and the dirs&files (nodekey, dapp, keystore, chaindata, nodes) do exist,
			if subdirMigrateErr := migrateToChainSubdirIfNecessary(ctx); subdirMigrateErr != nil {
				glog.Fatalf("%v: failed to migrate existing data to chain-specific subdir: %v", ErrDirectoryStructure, subdirMigrateErr)
			}
		}
		// Set default debug verbosity level.
		glog.SetV(5)

		// log.Println("Writing logs to ", logDir)
		// Turn on only file logging, disabling logging(T).toStderr and logging(T).alsoToStdErr
		glog.SetToStderr(false)
		glog.SetAlsoToStderr(false)

		// Set up file logging.
		logDir := filepath.Join(MustMakeChainDataDir(ctx), "log")
		if ctx.GlobalIsSet(aliasableName(LogDirFlag.Name, ctx)) {
			ld := ctx.GlobalString(aliasableName(LogDirFlag.Name, ctx))
			ldAbs, err := filepath.Abs(ld)
			if err != nil {
				glog.Fatalln(err)
			}
			logDir = ldAbs
		}
		// Ensure mkdir -p
		if e := os.MkdirAll(logDir, os.ModePerm); e != nil {
			return e
		}
		// Set log dir.
		// GOTCHA: There may be NO glog.V logs called before this is set.
		//   Otherwise everything will get all fucked and there will be no logs.
		glog.SetLogDir(logDir)

		if ctx.GlobalIsSet(DisplayFlag.Name) {
			i := ctx.GlobalInt(DisplayFlag.Name)
			if i > 3 {
				return fmt.Errorf("Error: --%s level must be 0 <= i <= 3, got: %d", DisplayFlag.Name, i)
			}
			glog.SetD(i)
		}

		if ctx.GlobalBool(NeckbeardFlag.Name) {
			glog.SetD(0)
			// Allow manual overrides
			if !ctx.GlobalIsSet(LogStatusFlag.Name) {
				ctx.Set(LogStatusFlag.Name, "sync=60") // set log-status interval
			}
			if !ctx.GlobalIsSet(VerbosityFlag.Name) {
				glog.SetV(5)
			}
			glog.SetAlsoToStderr(true)
		}
		// If --log-status not set, set default 60s interval
		if !ctx.GlobalIsSet(LogStatusFlag.Name) {
			ctx.Set(LogStatusFlag.Name, "sync=30")
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
			log.Fatalf("malformed %s flag value %q", aliasableName(TargetGasLimitFlag.Name, ctx), gasLimit)
		}

		// Set morden chain by default for dev mode.
		if ctx.GlobalBool(aliasableName(DevModeFlag.Name, ctx)) {
			if !ctx.GlobalIsSet(aliasableName(ChainIdentityFlag.Name, ctx)) {
				if e := ctx.Set(aliasableName(ChainIdentityFlag.Name, ctx), "morden"); e != nil {
					log.Fatalf("failed to set chain value: %v", e)
				}
			}
		}

		return nil
	}

	app.After = func(ctx *cli.Context) error {
		logger.Flush()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "Invalid command: %q. Please find `geth` usage below. \n", command)
		cli.ShowAppHelp(c)
		os.Exit(3)
	}
	return app
}

func main() {
	app := makeCLIApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// geth is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func geth(ctx *cli.Context) error {
	n := MakeSystemNode(Version, ctx)
	ethe := startNode(ctx, n)

	if ctx.GlobalString(LogStatusFlag.Name) != "off" {
		dispatchStatusLogs(ctx, ethe)
	}

	n.Wait()

	return nil
}
