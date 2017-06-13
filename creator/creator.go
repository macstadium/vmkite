package creator

import (
	"log"

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

func CloneVM(vs *vsphere.Session, params vsphere.VirtualMachineCloneParams) (*vsphere.VirtualMachine, error) {
	log.Printf("%#v", params)
	vm, err := vs.VirtualMachine(params.SrcName)
	if err != nil {
		return nil, err
	}
	clonedVm, err := vm.Clone(params)
	if err != nil {
		return nil, err
	}
	if err := clonedVm.PowerOn(); err != nil {
		return nil, err
	}
	return clonedVm, nil
}
