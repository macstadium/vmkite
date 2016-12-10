package main

import (
	"context"
	"encoding/json"
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

func main() {
	ctx := context.Background()

	state, err := getState(ctx)
	if err != nil {
		log.Fatal(err)
	}

	stj, _ := json.MarshalIndent(state, "", "  ")
	fmt.Printf("%s\n", stj)
}

// https://github.com/golang/build/blob/master/cmd/makemac/makemac.go
// State is the state of the world.
type State struct {
	mu sync.Mutex `json:"-"`

	Hosts  map[string]int    // IP address -> running vmkite VM count (including 0)
	VMHost map[string]string // "vmkite_8_host2b" => "10.0.0.0"
	HostIP map[string]string // "host-5" -> "10.0.0.0"
}

// https://github.com/golang/build/blob/master/cmd/makemac/makemac.go
// govc runs "govc <args...>" and ignores its output, unless there's an error.
func govc(ctx context.Context, args ...string) error {
	fmt.Fprintf(os.Stderr, "$ govc %v\n", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "govc", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("govc %s ...: %v, %s", args[0], err, out)
	}
	return nil
}

// https://github.com/golang/build/blob/master/cmd/makemac/makemac.go
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
			hostIP := st.HostIP[hostID]
			st.VMHost[name] = hostIP
			// if hostIP != "" && strings.HasPrefix(name, "vmkite_") {
			if hostIP != "" {
				st.Hosts[hostIP]++
			}
		}
	}

	return st, nil
}

// https://github.com/golang/build/blob/master/cmd/makemac/makemac.go
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
