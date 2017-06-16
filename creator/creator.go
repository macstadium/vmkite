package creator

import (
	"github.com/macstadium/vmkite/vsphere"
)

func CreateVM(vs *vsphere.Session, params vsphere.VirtualMachineCreationParams) (*vsphere.VirtualMachine, error) {
	vm, err := vs.CreateVM(params)
	if err != nil {
		return nil, err
	}
	if err := vm.PowerOn(); err != nil {
		return nil, err
	}
	return vm, nil
}
