// Copyright 2015 The go-ethereum Authors
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

// Contains the geth command usage template and generator.

package main

import (
	"io"

	"gopkg.in/urfave/cli.v1"
)

// AppHelpTemplate is the test template for the default, global app help topic.
var AppHelpTemplate = `NAME:
   {{.App.Name}} - {{.App.Usage}}

USAGE:
   {{.App.HelpName}} [options]{{if .App.Commands}} <command> [command options]{{end}} {{if .App.ArgsUsage}}{{.App.ArgsUsage}}{{else}}[arguments...]{{end}}

VERSION:
   {{.App.Version}}{{if .CommandAndFlagGroups}}

COMMANDS AND FLAGS:

{{range .CommandAndFlagGroups}}{{.Name}}
------------------------------------------------------------------------
  {{if .Commands}}{{range .Commands}}
    {{join .Names ", "}}{{ "\t" }}{{.Usage}}{{ if .Subcommands }}{{ "\n\n" }}{{end}}{{range .Subcommands}}{{"\t"}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}{{"\n"}}{{end}}{{end}}
  
  {{end}}{{range .Flags}}{{.}}
  {{end}}
{{end}}{{end}}{{if .App.Copyright }}

COPYRIGHT:
   {{.App.Copyright}}
   {{end}}
`

// flagGroup is a collection of commands and/or flags belonging to a single topic.
type flagGroup struct {
	Name     string
	Flags    []cli.Flag
	Commands []cli.Command
}

// AppHelpFlagAndCommandGroups is the application flags, grouped by functionality.
var AppHelpFlagAndCommandGroups = []flagGroup{
	{
		Name: "ETHEREUM",
		Commands: []cli.Command{
			importCommand,
			exportCommand,
			dumpChainConfigCommand,
			dumpCommand,
			rollbackCommand,
			recoverCommand,
			resetCommand,
		},
		Flags: []cli.Flag{
			DataDirFlag,
			ChainIdentityFlag,
			NetworkIdFlag,
			DevModeFlag,
			NodeNameFlag,
			FastSyncFlag,
			SlowSyncFlag,
			CacheFlag,
			LightKDFFlag,
			SputnikVMFlag,
			BlockchainVersionFlag,
		},
	},
	{
		Name: "ACCOUNT",
		Commands: []cli.Command{
			accountCommand,
			walletCommand,
			buildAddrTxIndexCommand,
		},
		Flags: []cli.Flag{
			KeyStoreDirFlag,
			UnlockedAccountFlag,
			PasswordFileFlag,
			AccountsIndexFlag,
			AddrTxIndexFlag,
			AddrTxIndexAutoBuildFlag,
		},
	},
	{
		Name: "API AND CONSOLE",
		Commands: []cli.Command{
			consoleCommand,
			attachCommand,
			javascriptCommand,
			apiCommand,
		},
		Flags: []cli.Flag{
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
			RPCCORSDomainFlag,
			JSpathFlag,
			ExecFlag,
			PreloadJSFlag,
		},
	},
	{
		Name: "NETWORKING",
		Flags: []cli.Flag{
			BootnodesFlag,
			ListenPortFlag,
			MaxPeersFlag,
			MaxPendingPeersFlag,
			NATFlag,
			NoDiscoverFlag,
			NodeKeyFileFlag,
			NodeKeyHexFlag,
		},
	},
	{
		Name: "MINER",
		Commands: []cli.Command{
			makeDagCommand,
		},
		Flags: []cli.Flag{
			MiningEnabledFlag,
			MinerThreadsFlag,
			MiningGPUFlag,
			AutoDAGFlag,
			EtherbaseFlag,
			TargetGasLimitFlag,
			GasPriceFlag,
			ExtraDataFlag,
		},
	},
	{
		Name: "GAS PRICE ORACLE",
		Flags: []cli.Flag{
			GpoMinGasPriceFlag,
			GpoMaxGasPriceFlag,
			GpoFullBlockRatioFlag,
			GpobaseStepDownFlag,
			GpobaseStepUpFlag,
			GpobaseCorrectionFactorFlag,
		},
	},
	{
		Name: "LOGGING AND DEBUGGING",
		Commands: []cli.Command{
			versionCommand,
			statusCommand,
			monitorCommand,
			makeMlogDocCommand,
			gpuInfoCommand,
			gpuBenchCommand,
		},
		Flags: []cli.Flag{
			VerbosityFlag,
			VModuleFlag,
			LogDirFlag,
			LogMaxSizeFlag,
			LogMinSizeFlag,
			LogMaxTotalSizeFlag,
			LogIntervalFlag,
			LogMaxAgeFlag,
			LogCompressFlag,
			LogStatusFlag,
			DisplayFlag,
			DisplayFormatFlag,
			NeckbeardFlag,
			MLogFlag,
			MLogDirFlag,
			MLogComponentsFlag,
			BacktraceAtFlag,
			MetricsFlag,
			FakePoWFlag,
		},
	},
	{
		Name: "EXPERIMENTAL",
		Flags: []cli.Flag{
			WhisperEnabledFlag,
			NatspecEnabledFlag,
			DisplayFlag,
			DisplayFormatFlag,
			NeckbeardFlag,
		},
	},
	{
		Name: "LEGACY",
		Commands: []cli.Command{
			upgradedbCommand,
		},
		Flags: []cli.Flag{
			TestNetFlag,
			Unused1,
		},
	},
	{
		Name: "MISCELLANEOUS",
		Flags: []cli.Flag{
			SolcPathFlag,
		},
	},
}

func init() {
	// Override the default app help template
	cli.AppHelpTemplate = AppHelpTemplate

	// Define a one shot struct to pass to the usage template
	type helpData struct {
		App                  interface{}
		CommandAndFlagGroups []flagGroup
	}
	// Override the default app help printer, but only for the global app help
	originalHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, tmpl string, data interface{}) {
		if tmpl == AppHelpTemplate {
			// Iterate over all the flags and add any uncategorized ones
			categorized := make(map[string]struct{})
			for _, group := range AppHelpFlagAndCommandGroups {
				for _, flag := range group.Flags {
					categorized[flag.String()] = struct{}{}
				}
			}
			uncategorized := []cli.Flag{}
			for _, flag := range data.(*cli.App).Flags {
				if _, ok := categorized[flag.String()]; !ok {
					uncategorized = append(uncategorized, flag)
				}
			}
			if len(uncategorized) > 0 {
				// Append all ungategorized options to the misc group
				miscs := len(AppHelpFlagAndCommandGroups[len(AppHelpFlagAndCommandGroups)-1].Flags)
				AppHelpFlagAndCommandGroups[len(AppHelpFlagAndCommandGroups)-1].Flags = append(AppHelpFlagAndCommandGroups[len(AppHelpFlagAndCommandGroups)-1].Flags, uncategorized...)

				// Make sure they are removed afterwards
				defer func() {
					AppHelpFlagAndCommandGroups[len(AppHelpFlagAndCommandGroups)-1].Flags = AppHelpFlagAndCommandGroups[len(AppHelpFlagAndCommandGroups)-1].Flags[:miscs]
				}()
			}
			// Render out custom usage screen
			originalHelpPrinter(w, tmpl, helpData{data, AppHelpFlagAndCommandGroups})
		} else {
			originalHelpPrinter(w, tmpl, data)
		}
	}
}
