package cmd

import (
	"context"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/runner"
	"github.com/macstadium/vmkite/vsphere"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var runParams struct {
	buildkiteApiToken   string
	buildkiteAgentToken string
	buildkiteOrg        string
	buildkitePipelines  []string
	concurrency         int
	apiListenOn         string
	apiTokenSecret      string
}

var vmParams = struct {
	vmPath              string
	vmClusterPath       string
	vmDS                string
	vmdkDS              string
	vmdkPath            string
	vmNetwork           string
	vmMemoryMB          int64
	vmNumCPUs           int32
	vmNumCoresPerSocket int32
	vmGuestId           string
	vmGuestInfo         map[string]string
}{
	vmGuestInfo: map[string]string{},
}

var vsphereParams vsphere.ConnectionParams

func ConfigureRun(app *kingpin.Application) {
	cmd := app.Command("run", "wait for Buildkite jobs, launch VMs")

	// run params

	cmd.Flag("buildkite-agent-token", "Buildkite Agent Token").
		Required().
		StringVar(&runParams.buildkiteAgentToken)

	cmd.Flag("buildkite-api-token", "Buildkite API Token").
		Required().
		StringVar(&runParams.buildkiteApiToken)

	cmd.Flag("buildkite-org", "Buildkite organization slug").
		Required().
		StringVar(&runParams.buildkiteOrg)

	cmd.Flag("buildkite-pipeline", "Limit to a specific buildkite pipelines").
		StringsVar(&runParams.buildkitePipelines)

	cmd.Flag("concurrency", "Limit how many concurrent jobs are run").
		Default("3").
		IntVar(&runParams.concurrency)

	cmd.Flag("api-listen", "The address and port for the api server to listen on").
		StringVar(&runParams.apiListenOn)

	cmd.Flag("api-token-secret", "The secret to use for generating api job auth tokens").
		StringVar(&runParams.apiTokenSecret)

	// vm params

	cmd.Flag("target-datastore", "name of datastore for new VM").
		Required().
		StringVar(&vmParams.vmDS)

	cmd.Flag("source-datastore", "name of datastore holding source image").
		Required().
		StringVar(&vmParams.vmdkDS)

	cmd.Flag("vm-cluster-path", "path to the cluster").
		Required().
		StringVar(&vmParams.vmClusterPath)

	cmd.Flag("vm-network-label", "name of network to connect VM to").
		Required().
		StringVar(&vmParams.vmNetwork)

	cmd.Flag("vm-memory-mb", "Specify the memory size in MB of the new virtual machine").
		Required().
		Int64Var(&vmParams.vmMemoryMB)

	cmd.Flag("vm-num-cpus", "Specify the number of the virtual CPUs of the new virtual machine").
		Required().
		Int32Var(&vmParams.vmNumCPUs)

	cmd.Flag("vm-num-cores-per-socket", "Number of cores used to distribute virtual CPUs among sockets in this virtual machine").
		Required().
		Int32Var(&vmParams.vmNumCoresPerSocket)

	cmd.Flag("vm-guest-id", "The guestid of the vm").
		Default("darwin14_64Guest").
		StringVar(&vmParams.vmGuestId)

	cmd.Flag("vm-guest-info", "A set of key=value params to pass to the vm").
		StringMapVar(&vmParams.vmGuestInfo)

	app.Flag("vm-path", "path to folder containing virtual machines").
		Required().
		StringVar(&vmParams.vmPath)

	// vsphere params

	app.Flag("vsphere-host", "vSphere hostname or IP address").
		Required().
		StringVar(&vsphereParams.Host)

	app.Flag("vsphere-user", "vSphere username").
		Required().
		StringVar(&vsphereParams.User)

	app.Flag("vsphere-pass", "vSphere password").
		Required().
		StringVar(&vsphereParams.Pass)

	app.Flag("vsphere-insecure", "vSphere certificate verification").
		Default("false").
		BoolVar(&vsphereParams.Insecure)

	cmd.Action(cmdRun)
}

func cmdRun(c *kingpin.ParseContext) error {
	vs, err := vsphere.NewSession(context.Background(), vsphereParams)
	if err != nil {
		return err
	}

	bk, err := buildkite.NewSession(runParams.buildkiteOrg, runParams.buildkiteApiToken)
	if err != nil {
		return err
	}

	r := runner.NewRunner(vs, bk, runner.Params{
		Concurrency:    runParams.concurrency,
		Pipelines:      runParams.buildkitePipelines,
		ApiListenOn:    runParams.apiListenOn,
		ApiTokenSecret: runParams.apiTokenSecret,
	})

	return r.Run(vsphere.VirtualMachineCreationParams{
		BuildkiteAgentToken: runParams.buildkiteAgentToken,
		ClusterPath:         vmParams.vmClusterPath,
		VirtualMachinePath:  vmParams.vmPath,
		DatastoreName:       vmParams.vmDS,
		MemoryMB:            vmParams.vmMemoryMB,
		Name:                "", // automatic
		NetworkLabel:        vmParams.vmNetwork,
		NumCPUs:             vmParams.vmNumCPUs,
		NumCoresPerSocket:   vmParams.vmNumCoresPerSocket,
		SrcDiskDataStore:    vmParams.vmdkDS,
		SrcDiskPath:         "", // per-job
		GuestInfo:           vmParams.vmGuestInfo,
	})
}
