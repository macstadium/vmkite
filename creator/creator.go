package creator

import (
	"fmt"
	"strings"
	"time"

	"github.com/macstadium/vmkite/vsphere"
)

func CreateVM(vs *vsphere.Session, params vsphere.VirtualMachineCreationParams) (*vsphere.VirtualMachine, error) {
	if params.Name == "" {
		params.Name = createMachineName(params)
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

func createMachineName(params vsphere.VirtualMachineCreationParams) string {
	return fmt.Sprintf(
		"vmkite-%s-%s",
		normalizeGuestID(params.GuestID),
		time.Now().Format("200612-150405"),
	)
}

func normalizeGuestID(id string) string {
	id = strings.Replace(id, "_", "-", -1)
	id = strings.Replace(id, "guest", "", -1)
	return strings.ToLower(id)
}
