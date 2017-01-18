package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lox/vmkite/vsphere"

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

	st := &state{}

	err = createVM(vs, st)
	if err != nil {
		return err
	}

	return nil
}

func createVM(vs *vsphere.Session, st *state) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	ts := time.Now().Format("200612-150405")
	name := fmt.Sprintf("vmkite-host-macOS_10_%d-%s", macOsMinor, ts)
	params := vsphere.VirtualMachineCreationParams{
		BuildkiteAgentToken: buildkiteAgentToken,
		ClusterPath:         vmClusterPath,
		DatastoreName:       vmDS,
		MacOsMinorVersion:   macOsMinor,
		MemoryMB:            vmMemoryMB,
		Name:                name,
		NetworkLabel:        vmNetwork,
		NumCPUs:             vmNumCPUs,
		NumCoresPerSocket:   vmNumCoresPerSocket,
		SrcDiskDataStore:    vmdkDS,
		SrcDiskPath:         vmdkPath,
	}
	vm, err := vs.CreateVM(params)
	if err != nil {
		return err
	}
	if err := vm.PowerOn(); err != nil {
		return err
	}
	return nil
}

// whichAInUse reports whether a VM is running on the provided hostIP named
// with suffix "a".
//
// st.mu must be held
func (st *state) whichAInUse(ip string) bool {
	suffix := fmt.Sprintf("-%s-a", ip) // vmkite-host-macOS_10_%d-%s-%s
	for _, vm := range st.VirtualMachines {
		if strings.HasSuffix(vm.Name, suffix) {
			return true
		}
	}
	return false
}
