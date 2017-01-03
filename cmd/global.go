package cmd

import kingpin "gopkg.in/alecthomas/kingpin.v2"

var (
	clusterPath string
	vmPath      string

	vsHost     string
	vsUser     string
	vsPass     string
	vsInsecure bool
)

func ConfigureGlobal(app *kingpin.Application) {
	app.Flag("vsphere-host", "vSphere hostname or IP address").
		Required().
		Envar("VS_HOST").
		StringVar(&vsHost)

	app.Flag("vsphere-user", "vSphere username").
		Required().
		Envar("VS_USER").
		StringVar(&vsUser)

	app.Flag("vsphere-pass", "vSphere password").
		Required().
		Envar("VS_PASS").
		StringVar(&vsPass)

	app.Flag("vsphere-insecure", "vSphere certificate verification").
		Default("false").
		Envar("VS_INSECURE").
		BoolVar(&vsInsecure)

	app.Flag("cluster-path", "path to the vSphere cluster").
		Default("/MacStadium - Vegas/host/XSERVE_Cluster").
		StringVar(&clusterPath)

	app.Flag("vm-path", "path to folder containing virtual machines").
		Default("/MacStadium - Vegas/vm").
		StringVar(&vmPath)
}
