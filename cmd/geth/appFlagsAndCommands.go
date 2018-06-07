package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// flagGroup is a collection of commands and/or flags belonging to a single topic.
type flagGroup struct {
	Name     string
	Flags    []cli.Flag
	Commands []cli.Command
}

type MarshalableConfig map[string]MarshalableConfigCategory
type MarshalableConfigCategory map[string]interface{}

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
			AppConfigFileFlag,
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

	// log.Println("args: ", ctx.Args()) // [dump-app-config --toml]

	args := ctx.Args()
	encType := ""

	if len(args) < 2 {
		encType = "toml"
	}
	if len(args) > 3 {
		log.Fatalln(ErrInvalidFlag)
	}
	if encType != "toml" {
		encType = args[1]
	}
	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
	}

	// get encoding
	// PTAL: eventually add more encoders, maybe; json, yaml
	if strings.Contains(encType, "toml") {
		encType = "toml"
	} else if strings.Contains(encType, "json") {
		log.Fatalln(errEncodingNotSupported)
	} else if strings.Contains(encType, "yaml") {
		log.Fatalln(errEncodingNotSupported)
	} else {
		log.Fatalln(errEncodingNotSupported)
	}

	// if no file given, just write to stderr
	if outFile == "" {
		if err := writeDefaultFlagsConfig(os.Stdout, encType); err != nil {
			log.Fatalln(err)
		}
	} else {
		// otherwise write to file
		c := filepath.Clean(outFile)
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

var errEncodingNotSupported = errors.New("encoding method not supported")

func writeDefaultFlagsConfig(w io.Writer, encodingType string) error {
	writeToml := func(w io.Writer) error {

		encoder := toml.NewEncoder(w)

		var c = make(MarshalableConfig)

		for _, g := range AppHelpFlagAndCommandGroups {

			cat := make(map[string]interface{})

			for _, f := range g.Flags {
				// don't set this, to avoid recursive configuration fuckups
				if f.GetName() == AppConfigFileFlag.Name {
					continue
				}
				// gotta do these damn switcher to get at the vals
				switch t := f.(type) {
				case cli.StringFlag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value
				case cli.BoolFlag:
					cat[strings.Split(t.Name, ",")[0]] = false
				case cli.BoolTFlag:
					cat[strings.Split(t.Name, ",")[0]] = true
				case cli.Float64Flag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value
				case cli.DurationFlag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value
				case cli.IntFlag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value
				case cli.IntSliceFlag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value
				case cli.StringSliceFlag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value
				case DirectoryFlag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value.String()
				case cli.GenericFlag:
					cat[strings.Split(t.Name, ",")[0]] = t.Value
				default:
					log.Fatalln("weird flag")
				}
			}
			c[g.Name] = cat
		}
		if err := encoder.Encode(c); err != nil {
			return err
		}
		return nil
	}
	switch encodingType {
	case "toml":
		return writeToml(w)
	}

	return errEncodingNotSupported
}

func readDefaultFlagsConfig(reader io.Reader, c *MarshalableConfig, encodingType string) error {
	if encodingType == "toml" {
		_, err := toml.DecodeReader(reader, c)
		return err
	}
	return errEncodingNotSupported
}

func mustReadAppConfigFromFile(ctx *cli.Context, ppath string) {
	ext := strings.TrimPrefix(filepath.Ext(ppath), ".")
	f, err := os.Open(ppath)
	if err != nil {
		log.Fatalln(err)
	}
	m := &MarshalableConfig{}
	if err := readDefaultFlagsConfig(f, m, ext); err != nil {
		log.Fatalln("could not decode configuration file: ", err)
	}
	glog.D(logger.Info).Infoln("Read config values from: ", logger.ColorGreen(ppath))
	for _, cat := range *m {
		for k, vv := range cat {
			if !ctx.GlobalIsSet(k) {
				ctx.GlobalSet(k, fmt.Sprintf("%v", vv)) // PTAL yuck...?
			}
		}
	}
}
