package cmd

import (
	"context"

	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	macOsMinor          = 11
	vmClusterPath       string
	vmDS                string
	vmdkDS              string
	vmdkPath            string
	vmNetwork           string
	vmMemoryMB          int64
	vmNumCPUs           int32
	vmNumCoresPerSocket int32
)

func ConfigureCreateVM(app *kingpin.Application) {
	cmd := app.Command("create-vm", "create a virtual machine")

	addCreateVMFlags(cmd)

	cmd.Flag("source-path", "path of source disk image").
		Required().
		StringVar(&vmdkPath)

	cmd.Flag("buildkite-agent-token", "Buildkite Agent Token").
		Required().
		StringVar(&buildkiteAgentToken)

	cmd.Action(cmdCreateVM)
}

func addCreateVMFlags(cmd *kingpin.CmdClause) {
	cmd.Flag("target-datastore", "name of datastore for new VM").
		Required().
		StringVar(&vmDS)

	cmd.Flag("source-datastore", "name of datastore holding source image").
		Required().
		StringVar(&vmdkDS)

	cmd.Flag("vm-cluster-path", "path to the cluster").
		Required().
		StringVar(&vmClusterPath)

	cmd.Flag("vm-network-label", "name of network to connect VM to").
		Required().
		StringVar(&vmNetwork)

	cmd.Flag("vm-memory-mb", "Specify the memory size in MB of the new virtual machine").
		Required().
		Int64Var(&vmMemoryMB)

	cmd.Flag("vm-num-cpus", "Specify the number of the virtual CPUs of the new virtual machine").
		Required().
		Int32Var(&vmNumCPUs)

	cmd.Flag("vm-num-cores-per-socket", "Number of cores used to distribute virtual CPUs among sockets in this virtual machine").
		Required().
		Int32Var(&vmNumCoresPerSocket)
}

func cmdCreateVM(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, connectionParams)
	if err != nil {
		return err
	}

	params := vsphere.VirtualMachineCreationParams{
		BuildkiteAgentToken: buildkiteAgentToken,
		ClusterPath:         vmClusterPath,
		DatastoreName:       vmDS,
		MacOsMinorVersion:   macOsMinor,
		MemoryMB:            vmMemoryMB,
		Name:                "",
		NetworkLabel:        vmNetwork,
		NumCPUs:             vmNumCPUs,
		NumCoresPerSocket:   vmNumCoresPerSocket,
		SrcDiskDataStore:    vmdkDS,
		SrcDiskPath:         vmdkPath,
	}

	err = creator.CreateVM(vs, params)
	if err != nil {
		return err
	}

	return nil
}
