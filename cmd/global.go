package cmd

import (
	"github.com/macstadium/vmkite/vsphere"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	clusterPath      string
	vmPath           string
	connectionParams vsphere.ConnectionParams
)

func ConfigureGlobal(app *kingpin.Application) {
	app.Flag("vsphere-host", "vSphere hostname or IP address").
		Required().
		StringVar(&connectionParams.Host)

	app.Flag("vsphere-user", "vSphere username").
		Required().
		StringVar(&connectionParams.User)

	app.Flag("vsphere-pass", "vSphere password").
		Required().
		StringVar(&connectionParams.Pass)

	app.Flag("vsphere-insecure", "vSphere certificate verification").
		Default("false").
		BoolVar(&connectionParams.Insecure)

	app.Flag("vm-path", "path to folder containing virtual machines").
		Required().
		StringVar(&vmPath)
}
