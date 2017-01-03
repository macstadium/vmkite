package cmd

import (
	"github.com/lox/vmkite/vsphere"
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
		Envar("VS_HOST").
		StringVar(&connectionParams.Host)

	app.Flag("vsphere-user", "vSphere username").
		Required().
		Envar("VS_USER").
		StringVar(&connectionParams.User)

	app.Flag("vsphere-pass", "vSphere password").
		Required().
		Envar("VS_PASS").
		StringVar(&connectionParams.Pass)

	app.Flag("vsphere-insecure", "vSphere certificate verification").
		Default("false").
		Envar("VS_INSECURE").
		BoolVar(&connectionParams.Insecure)

	app.Flag("cluster-path", "path to the vSphere cluster").
		Default("/MacStadium - Vegas/host/XSERVE_Cluster").
		StringVar(&clusterPath)

	app.Flag("vm-path", "path to folder containing virtual machines").
		Default("/MacStadium - Vegas/vm").
		StringVar(&vmPath)
}
