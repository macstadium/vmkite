package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/lox/vmkite/cmd"
)

func main() {
	run(os.Args[1:], os.Exit)
}

func run(args []string, exit func(code int)) {
	app := kingpin.New(
		"vmkite",
		"Manage VMware vSphere macOS VMs for CI builds",
	)

	app.Writer(os.Stdout)
	app.DefaultEnvars()
	app.Terminate(exit)

	cmd.ConfigureGlobal(app)

	cmd.ConfigureCreateVM(app)
	cmd.ConfigureDestroyVM(app)
	cmd.ConfigureRun(app)
	cmd.ConfigureStatus(app)

	kingpin.MustParse(app.Parse(args))
}
