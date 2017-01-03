package main

// Adapted frmom go/build's makemac.go:
// https://github.com/golang/build/blob/9412838986dd9e33/cmd/makemac/makemac.go

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/lox/vmkite/vsphere"
)

var clusterPath = "/MacStadium - Vegas/host/XSERVE_Cluster"
var vmPath = "/MacStadium - Vegas/vm"
var managedVMPrefix = "vmkite-"
var macOsMinor = 11
var vmDS = "PURE1-1"
var vmdkDS = "PURE1-1"
var vmdkPath = "vmkite-test-2/vmkite-test-2.vmdk"
var hostIPPrefix = "10.92.157."
var vmNetwork = "dvPortGroup-Private-1"
var vmMemoryMB int64 = 4096
var vmNumCPUs int32 = 4
var vmNumCoresPerSocket int32 = 1

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: vmkite [--status] [--create]\n")
	os.Exit(1)
}

var flagStatus = flag.Bool("status", false, "print status")
var flagCreate = flag.Bool("create", false, "create a VM")

type state struct {
	mu sync.Mutex `json:"-"`

	HostSystems        map[string]vsphere.HostSystem // HostID => HostSystem
	VirtualMachines    []vsphere.VirtualMachine
	HostManagedVMCount map[string]int // HostID => count
}

func main() {
	flag.Parse()
	if !(flag.NArg() == 0 && (*flagStatus || *flagCreate)) {
		usage()
	}

	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, vsphere.ConnectionParams{
		Host:     os.Getenv("VS_HOST"),
		User:     os.Getenv("VS_USER"),
		Pass:     os.Getenv("VS_PASS"),
		Insecure: os.Getenv("VS_INSECURE") == "true",
	})
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

	if *flagStatus {
		printState(st)
	}

	if *flagCreate {
		err = createVM(vs, st)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func printState(st *state) {
	stj, _ := json.MarshalIndent(st, "", "  ")
	fmt.Printf("%s\n", stj)
}

func createVM(vs *vsphere.Session, st *state) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	hs, hostWhich, err := st.pickHost()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("vmkite-macOS_10_%d-host-%s-%s", macOsMinor, hs.IP, hostWhich)

	params := vsphere.VirtualMachineCreationParams{
		DatastoreName:     vmDS,
		HostSystem:        hs,
		MacOsMinorVersion: macOsMinor,
		MemoryMB:          vmMemoryMB,
		Name:              name,
		NetworkLabel:      vmNetwork,
		NumCPUs:           vmNumCPUs,
		NumCoresPerSocket: vmNumCoresPerSocket,
		SrcDiskDataStore:  vmdkDS,
		SrcDiskPath:       vmdkPath,
	}
	if err := vs.CreateVM(params); err != nil {
		return err
	}
	return nil
}

// st.mu must be held.
func (st *state) pickHost() (hs vsphere.HostSystem, hostWhich string, err error) {
	for hostID, inUse := range st.HostManagedVMCount {
		hs = st.HostSystems[hostID]
		ip := hs.IP
		if !strings.HasPrefix(ip, hostIPPrefix) {
			continue
		}
		if inUse >= 2 {
			// Apple virtualization license policy.
			continue
		}
		hostWhich = "a" // unless in use
		if st.whichAInUse(ip) {
			hostWhich = "b"
		}
		return
	}
	err = errors.New("no usable host found")
	return
}

// whichAInUse reports whether a VM is running on the provided hostIP named
// with suffix "a".
//
// st.mu must be held
func (st *state) whichAInUse(ip string) bool {
	suffix := fmt.Sprintf("-host-%s-a", ip) // vmkite-macOS_10_%d-host-%s-%s
	for _, vm := range st.VirtualMachines {
		if strings.HasSuffix(vm.Name, suffix) {
			return true
		}
	}
	return false
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
