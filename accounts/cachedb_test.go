package accounts

import (
	"testing"
	"path/filepath"
	"github.com/ethereumproject/go-ethereum/common"
)

var (
	cachedbtestDir, _   = filepath.Abs(filepath.Join("testdata", "keystore"))
	cachedbtestAccounts = []Account{
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:    "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8",
			EncryptedKey: "{\"address\":\"7ef5a6135f1fd6a02593eedc869c6d41d934aef8\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"1d0839166e7a15b9c1333fc865d69858b22df26815ccf601b28219b6192974e1\",\"cipherparams\":{\"iv\":\"8df6caa7ff1b00c4e871f002cb7921ed\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"e5e6ef3f4ea695f496b643ebd3f75c0aa58ef4070e90c80c5d3fb0241bf1595c\"},\"mac\":\"6d16dfde774845e4585357f24bce530528bc69f4f84e1e22880d34fa45c273e5\"},\"id\":\"950077c7-71e3-4c44-a4a1-143919141ed4\",\"version\":3}",
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:    "aaa",
			EncryptedKey: "{\"address\":\"f466859ead1932d743d622cb74fc058882e8648a\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"cb664472deacb41a2e995fa7f96fe29ce744471deb8d146a0e43c7898c9ddd4d\",\"cipherparams\":{\"iv\":\"dfd9ee70812add5f4b8f89d0811c9158\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"0d6769bf016d45c479213990d6a08d938469c4adad8a02ce507b4a4e7b7739f1\"},\"mac\":\"bac9af994b15a45dd39669fc66f9aa8a3b9dd8c22cb16e4d8d7ea089d0f1a1a9\"},\"id\":\"472e8b3d-afb6-45b5-8111-72c89895099a\",\"version\":3}",
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:    "zzz",
			EncryptedKey: "{\"address\":\"289d485d9771714cce91d3393d764e1311907acc\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"faf32ca89d286b107f5e6d842802e05263c49b78d46eac74e6109e9a963378ab\",\"cipherparams\":{\"iv\":\"558833eec4a665a8c55608d7d503407d\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"d571fff447ffb24314f9513f5160246f09997b857ac71348b73e785aab40dc04\"},\"mac\":\"21edb85ff7d0dab1767b9bf498f2c3cb7be7609490756bd32300bb213b59effe\"},\"id\":\"3279afcf-55ba-43ff-8997-02dcc46a6525\",\"version\":3}",
		},
	}
)

func TestMain(m *testing.M) {
	m.Run()
}