// Copyright (c) 2017 ETCDEV Team

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

package process

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/machine/rpc/server"
	"os/signal"
	"syscall"
)

var (
	IpcFlag = cli.StringFlag{
		Name:  "ipc",
		Usage: "name to listen local rpc calls",
		Value: vm.DefaultVmIpc,
	}
	LoggingFlag = cli.IntFlag{
		Name:  "logging",
		Usage: "sets the logging level",
		Value: logger.Error,
	}
	SysStatFlag = cli.BoolFlag{
		Name:  "sysstat",
		Usage: "display system stats",
	}
)

func serve(newMachine func() (vm.Machine, error)) func(*cli.Context) error {
	return func(ctx *cli.Context) (err error) {
		var machine vm.Machine
		if machine, err = newMachine(); err == nil {
			if serv, err := server.NewServer(machine); err == nil {
				if err = serv.ServeIpc(ctx.String(IpcFlag.Name)); err == nil {
					sigc := make(chan os.Signal, 1)
					signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.Signal(21))
					<-sigc
					signal.Stop(sigc)
					serv.Close()
				}
			}
		}
		return
	}
}

func exec(ctx *cli.Context, run func(ctx *cli.Context) error) (err error) {

	glog.SetToStderr(true)
	glog.SetV(ctx.GlobalInt(LoggingFlag.Name))

	err = run(ctx)

	if ctx.GlobalBool(SysStatFlag.Name) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		fmt.Printf(`alloc:      %d
tot alloc:  %d
no. malloc: %d
heap alloc: %d
heap objs:  %d
num gc:     %d
`, mem.Alloc, mem.TotalAlloc, mem.Mallocs, mem.HeapAlloc, mem.HeapObjects, mem.NumGC)
	}

	return
}

func getUsage(newMachine func() (vm.Machine, error)) (string, error) {
	machine, err := newMachine()
	if err != nil {
		return "", err
	}
	tp := machine.Type()
	switch tp {
	case vm.ClassicVm:
		return "Classic Virtual Machine", nil
	case vm.OriginalVm:
		return "Classic Virtual Machine (RAW)", nil
	case vm.SputnikVm:
		return "Sputnik Virtual Machine", nil
	default:
		return machine.Name(), nil
	}
}

func Main(version string, newMachine func() (vm.Machine, error)) {
	var err error

	app := cli.NewApp()
	app.Version = version
	app.Name = filepath.Base(os.Args[0])
	app.Copyright = "Copyright (c) 2017, ETCDEV Team"
	app.Action = func(ctx *cli.Context) error { cli.ShowAppHelp(ctx); return nil }
	app.Flags = []cli.Flag{
		LoggingFlag,
		SysStatFlag,
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:   "serve",
			Usage:  `serve VM ipc server`,
			Action:  func(ctx *cli.Context) error { return exec(ctx,serve(newMachine)) },
			Flags: []cli.Flag{
				IpcFlag,
			},
		},
	}

	app.Usage, err = getUsage(newMachine)
	if err == nil {
		err = app.Run(os.Args)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
