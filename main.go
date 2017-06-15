package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/macstadium/vmkite/cmd"
)

var (
	Version string
)

func main() {
	run(os.Args[1:], os.Exit)
}

func run(args []string, exit func(code int)) {
	app := kingpin.New(
		"vmkite",
		"Spawn and manage ephemeral VMware VMs for Buildkite builds",
	)

	app.Version(Version)
	app.Writer(os.Stdout)
	app.DefaultEnvars()
	app.Terminate(exit)

	cmd.ConfigureGlobal(app)

	cmd.ConfigureCreateVM(app)
	cmd.ConfigureDestroyVM(app)
	cmd.ConfigureRun(app)

	kingpin.MustParse(app.Parse(args))
}
