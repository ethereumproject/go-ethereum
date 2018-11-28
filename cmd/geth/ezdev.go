package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
)

func mustSetCTXDefault(ctx *cli.Context, name, val string) {
	if err := ctx.GlobalSet(aliasableName(name, ctx), val); err != nil {
		log.Fatal(err)
	}
}

func setEZDevFlags(ctx *cli.Context) {
	mustSetCTXDefault(ctx, NoDiscoverFlag.Name, "true")
	mustSetCTXDefault(ctx, LightKDFFlag.Name, "true")
	mustSetCTXDefault(ctx, MiningEnabledFlag.Name, "true")
}

func setupEZDev(ctx *cli.Context, config *core.SufficientChainConfig) error {
	config.Include = []string{"dev_genesis.json"}

	cg := config.Genesis

	// Set original genesis to nil so no conflict between GenesisAlloc field and present Genesis obj.
	config.Genesis = nil
	cg.AllocFile = "dev_genesis_alloc.csv"

	accman := MakeAccountManager(ctx)
	data := []byte{}
	bal := "10000000000000000000000000000000"
	if len(accman.Accounts()) == 0 {
		glog.D(logger.Warn).Infoln("No existing EZDEV accounts found, creating 10")
		password := ""
		for i := 0; i < 10; i++ {
			acc, err := accman.NewAccount(password)
			if err != nil {
				return err
			}
			glog.D(logger.Warn).Infoln(acc.Address.Hex(), acc.File)
			// poor man's csv writer
			d := fmt.Sprintf(`"%s","%v"%c`, acc.Address.Hex(), bal, '\n')
			data = append(data, []byte(d)...)
		}
	} else {
		glog.D(logger.Warn).Infoln("Found existing keyfiles, using: ")
		for _, acc := range accman.Accounts() {
			d := fmt.Sprintf("%s,%v\n", acc.Address.Hex(), bal)
			glog.D(logger.Warn).Infoln(acc.Address.Hex(), acc.File)
			data = append(data, []byte(d)...)
		}
	}

	// marshal and write config json IFF it doesn't already exist
	chainFileP := filepath.Join(MustMakeChainDataDir(ctx), "chain.json")
	if _, err := os.Stat(chainFileP); err != nil && os.IsNotExist(err) {
		if err := config.WriteToJSONFile(chainFileP); err != nil {
			return err
		}
	}

	// marshal and write dev_genesis.json
	// this ugly structing only because genesis needs to be inside an object's brackets
	genesisFileP := filepath.Join(MustMakeChainDataDir(ctx), "dev_genesis.json")
	genC, err := json.MarshalIndent(struct {
		Genesis *core.GenesisDump `json:"genesis"`
	}{cg}, "", "    ")
	if err != nil {
		return fmt.Errorf("Could not marshal json from chain config: %v", err)
	}
	if err := ioutil.WriteFile(genesisFileP, genC, 0644); err != nil {
		return err
	}

	// write alloc file, ALWAYS, because these never change and it's just extra logic, even though it would seem more right to care if the file already exists or not
	genesisAllocFileP := filepath.Join(MustMakeChainDataDir(ctx), "dev_genesis_alloc.csv")
	ioutil.WriteFile(genesisAllocFileP, data, os.ModePerm)

	// again.. hacky.
	// 1. proves we're writing a valid config file setup
	// 2. let's us read the genesis alloc file (which uses nonexported hex type as map key; using this exported method is
	// at this point easier than refactoring to expose the genesis alloc stuff)
	cc, err := core.ReadExternalChainConfigFromFile(chainFileP)
	if err != nil {
		panic(err)
	}
	config.Genesis = cc.Genesis

	config.ChainConfig.Automine = true

	return nil
}
