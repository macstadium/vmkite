package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/lox/vmkite/vsphere"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	managedVMPrefix = "vmkite-"
)

func ConfigureStatus(app *kingpin.Application) {
	cmd := app.Command("status", "show status of cluster")

	cmd.Action(cmdStatus)
}

type state struct {
	mu sync.Mutex `json:"-"`

	HostSystems        map[string]vsphere.HostSystem // HostID => HostSystem
	VirtualMachines    []vsphere.VirtualMachine
	HostManagedVMCount map[string]int // HostID => count
}

func cmdStatus(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, connectionParams)
	if err != nil {
		log.Fatal(err)
	}

	st := &state{}

	if err = loadHostSystems(vs, st, clusterPath); err != nil {
		log.Fatal(err)
	}

	if err = loadVirtualMachines(vs, st, vmPath); err != nil {
		log.Fatal(err)
	}

	countManagedVMsPerHost(st, managedVMPrefix)

	printState(st)
	return nil
}

func printState(st *state) {
	stj, _ := json.MarshalIndent(st, "", "  ")
	fmt.Printf("%s\n", stj)
}

func loadHostSystems(vs *vsphere.Session, st *state, path string) error {
	results, err := vs.HostSystems(path)
	if err != nil {
		return err
	}
	st.HostSystems = make(map[string]vsphere.HostSystem)
	for _, hs := range results {
		st.HostSystems[hs.ID] = hs
	}
	return nil
}

func loadVirtualMachines(vs *vsphere.Session, st *state, path string) (err error) {
	path = fmt.Sprintf("%s/*", path)
	results, err := vs.VirtualMachines(path)
	if err == nil {
		st.VirtualMachines = results
	}
	return
}

func countManagedVMsPerHost(st *state, prefix string) {
	st.HostManagedVMCount = make(map[string]int)
	for _, hs := range st.HostSystems {
		st.HostManagedVMCount[hs.ID] = 0
	}
	for _, vm := range st.VirtualMachines {
		if strings.HasPrefix(vm.Name, prefix) {
			st.HostManagedVMCount[vm.HostSystemID]++
		}
	}
}
