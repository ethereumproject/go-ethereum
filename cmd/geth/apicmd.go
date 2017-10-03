// Copyright 2017 The go-ethereum Authors
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
package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/node"
	"github.com/ethereumproject/go-ethereum/rpc"
)

var (
	apiCommand = cli.Command{
		Action:          execAPI,
		Name:            "api",
		Usage:           "Run any API command",
		SkipFlagParsing: true,
		Description: `
The api command allows to run any method defined in any API module.
`,
	}
)

func execAPI(ctx *cli.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	if err := verifyArguments(ctx, client); err != nil {
		return err
	}

	result, err := callRPC(ctx, client)
	if err != nil {
		return err
	}
	return prettyPrint(result)
}

func getClient(ctx *cli.Context) (rpc.Client, error) {
	chainDir := MustMakeChainDataDir(ctx)
	var uri = "ipc:" + node.DefaultIPCEndpoint(chainDir)
	return rpc.NewClient(uri)
}

func verifyArguments(ctx *cli.Context, client rpc.Client) error {
	modules, err := client.SupportedModules()
	if err != nil {
		return err
	}

	module := ctx.Args()[0]
	if _, ok := modules[module]; !ok {
		return fmt.Errorf("Unknown API module: %s", module)
	}

	return nil
}

func callRPC(ctx *cli.Context, client rpc.Client) (interface{}, error) {
	var (
		module = ctx.Args()[0]
		method = ctx.Args()[1]
		args   = ctx.Args()[2:]
	)
	req := map[string]interface{}{
		"id":      new(int64),
		"method":  module + "_" + method,
		"jsonrpc": "2.0",
		"params":  []interface{}{args},
	}

	if err := client.Send(req); err != nil {
		return nil, err
	}

	var res rpc.JSONSuccessResponse
	if err := client.Recv(&res); err != nil {
		return nil, err
	}
	if res.Result != nil {
		return res.Result, nil
	}

	return nil, errors.New("No API response")
}

func prettyPrint(result interface{}) error {
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}
