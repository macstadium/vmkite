package creator

import (
	"fmt"
	"strings"
	"time"

	"github.com/macstadium/vmkite/vsphere"
)

func CreateVM(vs *vsphere.Session, params vsphere.VirtualMachineCreationParams) (*vsphere.VirtualMachine, error) {
	if params.Name == "" {
		ts := time.Now().Format("200612-150405")
		t := strings.ToLower(strings.Replace(params.GuestType, "_", "-"))
		params.Name = fmt.Sprintf("vmkite-%s-%s", t, ts)
	}
	vm, err := vs.CreateVM(params)
	if err != nil {
		return nil, err
	}
	if err := vm.PowerOn(); err != nil {
		return nil, err
	}
	return vm, nil
}
