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

	finder, err := createFinder(ctx, c)
	if err != nil {
		log.Fatal(err)
	}

	state, err := getState(ctx, finder)
	if err != nil {
		log.Fatal(err)
	}

	if *flagStatus {
		printState(state)
	}

	if *flagCreate {
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

func createFinder(ctx context.Context, c *govmomi.Client) (*find.Finder, error) {
	log.Println("NewFinder()")
	finder := find.NewFinder(c.Client, true)
	log.Println("finder.DefaultDatacenter()")
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("finder.SetDatacenter(%v)", dc)
	finder.SetDatacenter(dc)
	return finder, nil
}

type State struct {
	mu sync.Mutex `json:"-"`

	Hosts  map[string]int    // IP address -> running vmkite VM count (including 0)
	VMHost map[string]string // "vmkite-macOS_10_11-host-10.1.2.12-b" => "10.1.2.12"
	HostIP map[string]string // "host-5" -> "10.1.2.15"
}

func printState(state *State) {
	stj, _ := json.MarshalIndent(state, "", "  ")
	fmt.Printf("%s\n", stj)
}

func createVM(ctx context.Context, st *State) error {
	log.Println("creating VM...")

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

	log.Println("creating VM", name)

	if err := govc(ctx, "vm.create",
		"-m", "4096",
		"-c", "6",
		"-on=false",
		"-net", vmNetwork,
		"-g", guestType,
		"-ds", vmDS,
		"-host.ip", hostIp,
		name,
	); err != nil {
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

func getState(ctx context.Context, finder *find.Finder) (*State, error) {
	st := &State{
		VMHost: make(map[string]string),
		Hosts:  make(map[string]int),
		HostIP: make(map[string]string),
	}

	log.Printf("finder.HostSystemList(%v)", clusterPath)
	hostSystems, err := finder.HostSystemList(ctx, clusterPath)
	if err != nil {
		return nil, err
	}
	log.Printf("host systems:")
	for _, hs := range hostSystems {
		ip := hs.Name()
		hostID := hs.Reference().Value
		log.Printf("  - %v: %v", hostID, ip)
		st.Hosts[ip] = 0
		st.HostIP[hostID] = ip
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
