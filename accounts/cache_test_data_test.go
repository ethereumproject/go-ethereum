package accounts

import (
	"github.com/eth-classic/go-ethereum/common"
	"path/filepath"
)

var (
	cachetestDir, _   = filepath.Abs(filepath.Join("testdata", "keystore"))
	cachetestAccounts = []Account{
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:    filepath.Join(cachetestDir, "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:    filepath.Join(cachetestDir, "aaa"),
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:    filepath.Join(cachetestDir, "zzz"),
		},
	}
)
