package main

import (
	"gopkg.in/urfave/cli.v1"
)

func setAppCommands(app *cli.App, cmds []cli.Command) {
	app.Commands = cmds
}

var appCmds = []cli.Command{
	importCommand,
	exportCommand,
	dumpChainConfigCommand,
	upgradedbCommand,
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
	makeDagCommand,
	gpuInfoCommand,
	gpuBenchCommand,
	versionCommand,
	makeMlogDocCommand,
	buildAddrTxIndexCommand,
}

var makeDagCommand = cli.Command{
	Action:  makedag,
	Name:    "make-dag",
	Aliases: []string{"makedag"},
	Usage:   "Generate ethash dag (for testing)",
	Description: `
		The makedag command generates an ethash DAG in /tmp/dag.

		This command exists to support the system testing project.
		Regular users do not need to execute it.
				`,
}

var gpuInfoCommand = cli.Command{
	Action:  gpuinfo,
	Name:    "gpu-info",
	Aliases: []string{"gpuinfo"},
	Usage:   "GPU info",
	Description: `
	Prints OpenCL device info for all found GPUs.
			`,
}

var gpuBenchCommand = cli.Command{
	Action:  gpubench,
	Name:    "gpu-bench",
	Aliases: []string{"gpubench"},
	Usage:   "Benchmark GPU",
	Description: `
	Runs quick benchmark on first GPU found.
			`,
}

var versionCommand = cli.Command{
	Action: version,
	Name:   "version",
	Usage:  "Print ethereum version numbers",
	Description: `
	The output of this command is supposed to be machine-readable.
			`,
}

var makeMlogDocCommand = cli.Command{
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
}
