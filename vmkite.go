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
	"os/exec"
	"path"
	"strings"
	"sync"
)

var clusterPath string = "/MacStadium - Vegas/host/XSERVE_Cluster"
var vmFolder string = "/MacStadium - Vegas/vm"
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

	state, err := getState(ctx)
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

// getStat queries govc to find the current state of the hosts and VMs.
func getState(ctx context.Context) (*State, error) {
	st := &State{
		VMHost: make(map[string]string),
		Hosts:  make(map[string]int),
		HostIP: make(map[string]string),
	}

	var hosts elementList
	log.Println("Getting cluster status")
	if err := govcJSONDecode(ctx, &hosts, "ls", "-json", clusterPath); err != nil {
		return nil, fmt.Errorf("Reading %s: %v", clusterPath, err)
	}
	for _, h := range hosts.Elements {
		if h.Object.Self.Type == "HostSystem" {
			ip := path.Base(h.Path)
			st.Hosts[ip] = 0
			st.HostIP[h.Object.Self.Value] = ip
		}
	}

	var vms elementList
	log.Println("Getting VM status")
	if err := govcJSONDecode(ctx, &vms, "ls", "-json", vmFolder); err != nil {
		return nil, fmt.Errorf("Reading vmFolder: %v", vmFolder, err)
	}
	for _, h := range vms.Elements {
		if h.Object.Self.Type == "VirtualMachine" {
			name := path.Base(h.Path)
			hostID := h.Object.Runtime.Host.Value
			hostIp := st.HostIP[hostID]
			st.VMHost[name] = hostIp
			// if hostIp != "" && strings.HasPrefix(name, "vmkite_") {
			if hostIp != "" {
				st.Hosts[hostIp]++
			}
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

// govcJSONDecode runs "govc <args...>" and decodes its JSON output into dst.
func govcJSONDecode(ctx context.Context, dst interface{}, args ...string) error {
	cmd := exec.CommandContext(ctx, "govc", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	err = json.NewDecoder(stdout).Decode(dst)
	cmd.Process.Kill() // usually unnecessary
	if werr := cmd.Wait(); werr != nil && err == nil {
		err = werr
	}
	return err
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
