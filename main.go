package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/lox/vmkite/cmd"
)

var (
	Version = "dev"
)

func main() {
	run(os.Args[1:], os.Exit)
}

func run(args []string, exit func(code int)) {
	app := kingpin.New(
		"vmkite",
		"Manage VMware vSphere macOS VMs for CI builds",
	)

	app.Version(Version)
	app.Writer(os.Stdout)
	app.DefaultEnvars()
	app.Terminate(exit)

	cmd.ConfigureGlobal(app)

	cmd.ConfigureStatus(app)
	cmd.ConfigureCreateVirtualMachine(app)

	kingpin.MustParse(app.Parse(args))
}
