package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	"gopkg.in/urfave/cli.v1"
	"io"
	"log"
	"os"
	"path/filepath"
)

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
			DocRootFlag,
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
			PprofFlag,
			PprofIntervalFlag,
		},
	},
	{
		Name: "EXPERIMENTAL",
		Flags: []cli.Flag{
			WhisperEnabledFlag,
			NatspecEnabledFlag,
		},
		Commands: []cli.Command{
			cmdDumpAppConfig,
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

var cmdDumpAppConfig = cli.Command{
	Action: func(ctx *cli.Context) error {
		return nil
	},
	Name:  "dump-app-config",
	Usage: `Dump default geth node configuration options.`,
	Description: `
	Dump default configuration values.

	If a filepath is provided as the first argument, the configuration data will be written there.
	If no argument is provided, the configuration will be echoed to stdout.
		`,
	Flags: []cli.Flag{
		cli.BoolTFlag{
			Name:  "toml",
			Usage: "Writes toml configuration defaults.",
		},
	},
}

func dumpAppConfig(ctx *cli.Context) error {
	// get encoding
	encType := ""
	if ctx.IsSet("toml") {
		encType = "toml"
		// TODO(whilei): add more encoders, maybe; json, yaml
	} else if ctx.IsSet("json") {
		log.Fatalln(errEncoderNotSupported)
	} else if ctx.IsSet("yaml") {
		log.Fatalln(errEncoderNotSupported)
	} else {
		encType = "toml"
	}

	// if no file given, just write to stderr
	if ctx.NArg() == 0 {
		if err := writeDefaultFlagsConfig(os.Stdout, encType); err != nil {
			log.Fatalln(err)
		}
	} else {
		// otherwise write to file
		c := filepath.Clean(ctx.Args()[0])
		f, err := os.Create(c)
		if err != nil && !os.IsExist(err) {
			log.Fatalln("could not create file:", err)
			return nil
		}
		if err := writeDefaultFlagsConfig(f, encType); err != nil {
			log.Fatalln(err)
		}
		return f.Close()
	}
	return nil
}

var errEncoderNotSupported = errors.New("encoding method not supported")

func writeDefaultFlagsConfig(w io.Writer, encodingType string) error {
	writeToml := func(w io.Writer) error {
		encoder := toml.NewEncoder(w)
		var flagGroupsMetaStruct = struct {
			Config []flagGroup
		}{}
		c := []flagGroup{}
		for _, g := range AppHelpFlagAndCommandGroups {
			imitGroup := flagGroup{
				Name: g.Name,
			}
			fs := imitGroup.Flags
			for _, f := range g.Flags {
				switch t := f.(type) {
				case cli.StringFlag:
					fs = append(fs, cli.StringFlag{
						Name:  t.Name,
						Value: t.Value,
					})
				case cli.BoolFlag:
					fs = append(fs, cli.BoolFlag{
						Name: t.Name,
						// Value: false, // FIXME(whilei)
					})
				case cli.BoolTFlag:
					fs = append(fs, cli.BoolTFlag{
						Name: t.Name,
					})
				case cli.Float64Flag:
					fs = append(fs, cli.Float64Flag{
						Name:  t.Name,
						Value: t.Value,
					})
				case cli.DurationFlag:
					fs = append(fs, cli.DurationFlag{
						Name:  t.Name,
						Value: t.Value,
					})
				case cli.IntFlag:
					fs = append(fs, cli.IntFlag{
						Name:  t.Name,
						Value: t.Value,
					})
				case cli.IntSliceFlag:
					fs = append(fs, cli.IntSliceFlag{
						Name:  t.Name,
						Value: t.Value,
					})
				case DirectoryFlag:
					fs = append(fs, DirectoryFlag{
						Name:  t.Name,
						Value: t.Value,
					})
				case cli.GenericFlag:
					fs = append(fs, cli.GenericFlag{
						Name:  t.Name,
						Value: t.Value,
					})
				default:
					log.Fatalln("weird flag")
				}
			}
			c = append(c)
		}
		flagGroupsMetaStruct.Config = c
		if err := encoder.Encode(flagGroupsMetaStruct); err != nil {
			return err
		}
		return nil
	}
	switch encodingType {
	case "toml":
		return writeToml(w)
	}

	return errEncoderNotSupported
}
