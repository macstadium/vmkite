package cmd

import (
	"github.com/macstadium/vmkite/vsphere"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var globalParams struct {
	connectionParams vsphere.ConnectionParams
}

func ConfigureGlobal(app *kingpin.Application) {
	app.Flag("vsphere-host", "vSphere hostname or IP address").
		Required().
		StringVar(&globalParams.connectionParams.Host)

	app.Flag("vsphere-user", "vSphere username").
		Required().
		StringVar(&globalParams.connectionParams.User)

	app.Flag("vsphere-pass", "vSphere password").
		Required().
		StringVar(&globalParams.connectionParams.Pass)

	app.Flag("vsphere-insecure", "vSphere certificate verification").
		Default("false").
		BoolVar(&globalParams.connectionParams.Insecure)
}
