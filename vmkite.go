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
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

var clusterPath string = "/MacStadium - Vegas/host/XSERVE_Cluster"
var vmPath string = "/MacStadium - Vegas/vm"
var managedVmPrefix string = "vmkite-"
var macOsMinor int = 11
var vmDS string = "PURE1-1"
var vmdkDS string = "PURE1-1"
var vmdkPath string = "vmkite-test-2/vmkite-test-2.vmdk"
var hostIpPrefix string = "10.92.157."
var vmNetwork string = "dvPortGroup-Private-1"
var vmMemoryMB int64 = 4096
var vmNumCPUs int32 = 4
var vmNumCoresPerSocket int32 = 1

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: makemac <macos_minor_version>\n")
	os.Exit(1)
}

var flagStatus = flag.Bool("status", false, "print status")
var flagCreate = flag.Bool("create", false, "create a VM")

func main() {
	flag.Parse()
	if !(flag.NArg() == 0 && (*flagStatus || *flagCreate)) {
		usage()
	}

	ctx := context.Background()

	c, err := vsphereLogin(ctx)
	if err != nil {
		log.Fatal(err)
	}

	state, err := getState(ctx, c)
	if err != nil {
		log.Fatal(err)
	}

	if *flagStatus {
		printState(state)
	}

	if *flagCreate {
		log.Println("createVM()")
		err = createVM(ctx, state)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func vsphereLogin(ctx context.Context) (*govmomi.Client, error) {
	host := os.Getenv("VS_HOST")
	user := os.Getenv("VS_USER")
	pass := os.Getenv("VS_PASS")
	insecure := os.Getenv("VS_INSECURE") == "true"
	if host == "" || user == "" || pass == "" {
		return nil, errors.New("VS_HOST, VS_USER, VS_PASS vSphere details required")
	}
	u, err := url.Parse(fmt.Sprintf("https://%s/sdk", host))
	if err != nil {
		return nil, err
	}
	log.Printf("NewClient(%s) user:%s", u.String(), user)
	u.User = url.UserPassword(user, pass)
	c, err := govmomi.NewClient(ctx, u, insecure)
	if err != nil {
		log.Fatal(err)
	}
	return c, nil
}

type State struct {
	mu     sync.Mutex   `json:"-"`
	finder *find.Finder `json:"-"`

	Datacenter *object.Datacenter
	Datastore  *object.Datastore

	Hosts  map[string]int    // IP address -> running vmkite VM count (including 0)
	VMHost map[string]string // "vmkite-macOS_10_11-host-10.1.2.12-b" => "10.1.2.12"
	HostIP map[string]string // "host-5" -> "10.1.2.15"

	HostSystems map[string]*object.HostSystem // "host-5" => object
	HostIDS     []string                      // "host-5", ...
}

func printState(state *State) {
	stj, _ := json.MarshalIndent(state, "", "  ")
	fmt.Printf("%s\n", stj)
}

func createVM(ctx context.Context, st *State) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	guestType, err := guestTypeForMinorVersion(macOsMinor)
	if err != nil {
		return err
	}

	hostIp, hostWhich, err := st.pickHost()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("vmkite-macOS_10_%d-host-%s-%s", macOsMinor, hostIp, hostWhich)

	dcFolders, err := st.Datacenter.Folders(ctx)
	if err != nil {
		return err
	}

	deviceSpecs := []types.BaseVirtualDeviceConfigSpec{}
	nd, err := createNetworkDevice(ctx, st, vmNetwork)
	if err != nil {
		return err
	}
	deviceSpecs = append(deviceSpecs, nd)

	scsi, err := object.SCSIControllerTypes().CreateSCSIController("scsi")
	if err != nil {
		return err
	}
	deviceSpecs = append(deviceSpecs, &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
		Device:    scsi,
	})

	fileInfo := &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", st.Datastore.Name()),
	}

	configSpec := types.VirtualMachineConfigSpec{
		DeviceChange:      deviceSpecs,
		Files:             fileInfo,
		GuestId:           guestType,
		MemoryMB:          vmMemoryMB,
		Name:              name,
		NumCPUs:           vmNumCPUs,
		NumCoresPerSocket: vmNumCoresPerSocket,
	}

	hostID := st.HostIDS[0] // TODO: use dynamically picked host
	hostSystem := st.HostSystems[hostID]
	resourcePool, err := hostSystem.ResourcePool(ctx)
	if err != nil {
		return err
	}

	log.Printf("CreateVM %s on %s (%s)", name, hostID, hostIp)

	task, err := dcFolders.VmFolder.CreateVM(ctx, configSpec, resourcePool, hostSystem)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err := govc(ctx, "vm.destroy", name)
			if err != nil {
				log.Printf("failed to destroy %v: %v", name, err)
			}
		}
	}()
	log.Println("waiting for CreateVM_Task")
	err = task.Wait(ctx)
	if err != nil {
		return err
	}

	if err := govc(ctx, "vm.change",
		"-e", "smc.present=TRUE",
		"-e", "ich7m.present=TRUE",
		"-e", "firmware=efi",
		"-e", "guestinfo.name="+name,
		"-vm", name,
	); err != nil {
		return err
	}

	if err := govc(ctx, "device.usb.add", "-vm", name); err != nil {
		return err
	}

	if err := govc(ctx, "vm.disk.attach",
		"-vm", name,
		"-link=true",
		"-persist=false",
		"-ds", vmdkDS,
		"-disk", vmdkPath,
	); err != nil {
		return err
	}

	if err := govc(ctx, "vm.power", "-on", name); err != nil {
		return err
	}
	log.Printf("Success.")
	return nil

}

// st.mu must be held.
func (st *State) pickHost() (hostIp string, hostWhich string, err error) {
	for ip, inUse := range st.Hosts {
		if !strings.HasPrefix(ip, hostIpPrefix) {
			continue
		}
		if inUse >= 2 {
			// Apple virtualization license policy.
			continue
		}
		hostIp = ip
		hostWhich = "a" // unless in use
		if st.whichAInUse(hostIp) {
			hostWhich = "b"
		}
		return
	}
	return "", "", errors.New("no usable host found")
}

// whichAInUse reports whether a VM is running on the provided hostIp named
// with suffix "a".
//
// st.mu must be held
func (st *State) whichAInUse(ip string) bool {
	suffix := fmt.Sprintf("-host-%s-a", ip) // vmkite-macOS_10_%d-host-%s-%s
	for name := range st.VMHost {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// govc runs "govc <args...>" and ignores its output, unless there's an error.
func govc(ctx context.Context, args ...string) error {
	log.Printf("govc %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "govc", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("govc %s ...: %v, %s", args[0], err, out)
	}
	return nil
}

func getState(ctx context.Context, c *govmomi.Client) (*State, error) {
	finder := find.NewFinder(c.Client, true)

	log.Println("finder.DefaultDatacenter()")
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("  - %s", dc)
	finder.SetDatacenter(dc)

	log.Printf("finder.Datastore(%v)", vmDS)
	ds, err := finder.Datastore(ctx, vmDS)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("  - %s", ds)

	st := &State{
		finder:      finder,
		Datacenter:  dc,
		Datastore:   ds,
		VMHost:      make(map[string]string),
		Hosts:       make(map[string]int),
		HostIP:      make(map[string]string),
		HostSystems: make(map[string]*object.HostSystem),
		HostIDS:     make([]string, 0),
	}

	log.Printf("finder.HostSystemList(%v)", clusterPath)
	hostSystems, err := finder.HostSystemList(ctx, clusterPath)
	if err != nil {
		return nil, err
	}
	for _, hs := range hostSystems {
		ip := hs.Name()
		hostID := hs.Reference().Value
		log.Printf("  - %v: %v", hostID, ip)
		st.Hosts[ip] = 0
		st.HostIP[hostID] = ip
		st.HostSystems[hostID] = hs
		st.HostIDS = append(st.HostIDS, hostID)
	}

	path := fmt.Sprintf("%s/*", vmPath)
	log.Printf("finder.VirtualMachineList(%v)", vmPath)
	virtualMachines, err := finder.VirtualMachineList(ctx, path)
	if err != nil {
		return nil, err
	}
	for _, vm := range virtualMachines {
		name := vm.Name()
		hs, err := vm.HostSystem(ctx)
		if err != nil {
			return nil, err
		}
		hostID := hs.Reference().Value
		hostIp := st.HostIP[hostID]
		log.Printf("  - %v on %v (%v)", name, hostID, hostIp)
		st.VMHost[name] = hostIp
		if hostIp != "" && strings.HasPrefix(name, managedVmPrefix) {
			st.Hosts[hostIp]++
		}
	}

	return st, nil
}

// objRef is a VMWare "Managed Object Reference".
type objRef struct {
	Type  string // e.g. "VirtualMachine"
	Value string // e.g. "host-12"
}

type elementList struct {
	Elements []*elementJSON `json:"elements"`
}

type elementJSON struct {
	Path   string
	Object struct {
		Self    objRef
		Runtime struct {
			Host objRef // for VMs; not present otherwise
		}
	}
}

func guestTypeForMinorVersion(minor int) (t string, err error) {
	switch minor {
	case 8:
		t = "darwin12_64Guest"
	case 9:
		t = "darwin13_64Guest"
	case 10, 11:
		t = "darwin14_64Guest"
	default:
		err = fmt.Errorf("unknown VM guest type for macOS 10.%d", minor)
	}
	return
}

func createNetworkDevice(ctx context.Context, state *State, label string) (*types.VirtualDeviceConfigSpec, error) {
	f := state.finder
	network, err := f.Network(ctx, "*"+label)
	if err != nil {
		return nil, err
	}

	backing, err := network.EthernetCardBackingInfo(ctx)
	if err != nil {
		return nil, err
	}

	return &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
		Device: &types.VirtualE1000{
			types.VirtualEthernetCard{
				VirtualDevice: types.VirtualDevice{
					Key:     -1,
					Backing: backing,
				},
				AddressType: string(types.VirtualEthernetCardMacTypeGenerated),
			},
		},
	}, nil
}
