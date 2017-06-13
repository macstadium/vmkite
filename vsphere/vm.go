package vsphere

import (
	"fmt"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// VirtualMachine wraps govmomi's object.VirtualMachine
type VirtualMachine struct {
	vs *Session
	mo *object.VirtualMachine

	Name string
}

func (vm *VirtualMachine) Destroy(powerOff bool) error {
	vs := vm.vs

	if powerOff {
		poweredOn, err := vm.IsPoweredOn()
		if err != nil {
			return err
		}
		if poweredOn {
			vm.PowerOff()
		}
	}

	debugf("vm.Destroy(%s)", vm.Name)
	task, err := vm.mo.Destroy(vs.ctx)
	if err != nil {
		return err
	}
	debugf("waiting for Destroy %v", task)
	if err := task.Wait(vs.ctx); err != nil {
		return err
	}
	return nil
}

func (vm *VirtualMachine) IsPoweredOn() (bool, error) {
	vs := vm.vs
	state, err := vm.mo.PowerState(vs.ctx)
	if err != nil {
		return false, err
	}
	return state == "poweredOn", nil
}

func (vm *VirtualMachine) PowerOff() error {
	vs := vm.vs
	debugf("wm.PowerOff(%s)", vm.Name)
	task, err := vm.mo.PowerOff(vs.ctx)
	if err != nil {
		return err
	}
	debugf("waiting for PowerOff %v", task)
	if err := task.Wait(vs.ctx); err != nil {
		return err
	}
	return nil
}

func (vm *VirtualMachine) PowerOn() error {
	vs := vm.vs
	debugf("vm.PowerOn(%s)", vm.Name)
	task, err := vm.mo.PowerOn(vs.ctx)
	if err != nil {
		return err
	}
	debugf("waiting for PowerOn %v", task)
	if err := task.Wait(vs.ctx); err != nil {
		return err
	}
	return nil
}

func (vm *VirtualMachine) CreateSnapshot(name, desc string, memory, quiesce bool) error {
	vs := vm.vs
	debugf("vm.CreateSnapshot(%s)", vm.Name)
	task, err := vm.mo.CreateSnapshot(vs.ctx, name, desc, memory, quiesce)
	if err != nil {
		return err
	}

	debugf("waiting for CreateSnapshot %v", task)
	if err := task.Wait(vs.ctx); err != nil {
		return err
	}
	return nil
}

// Clone clones an existing VM with the CloneVM api
func (vm *VirtualMachine) Clone(params VirtualMachineCloneParams) (*VirtualMachine, error) {
	vs := vm.vs
	cloneSpec, err := createCloneSpec(vm, params)
	if err != nil {
		return nil, err
	}

	folder, err := vs.vmFolder()
	if err != nil {
		return nil, err
	}

	debugf("vm.Clone() %s from %s", params.Name, params.SrcName)
	task, err := vm.mo.Clone(vs.ctx, folder, params.Name, cloneSpec)
	if err != nil {
		return nil, err
	}

	debugf("waiting for CloneVM %v", task)
	if err := task.Wait(vs.ctx); err != nil {
		return nil, err
	}

	clonedVM, err := vs.VirtualMachine(folder.InventoryPath + "/" + params.Name)
	if err != nil {
		return nil, err
	}

	configSpec := createCloneReconfigureSpec(params)
	if err := clonedVM.Reconfigure(configSpec); err != nil {
		if destroyErr := clonedVM.Destroy(false); destroyErr != nil {
			return nil, fmt.Errorf("Failed to set GuestInfo: %v and also failed to destroy VM: %v", err, destroyErr)
		}
		return nil, err
	}

	return clonedVM, nil
}

func (vm *VirtualMachine) Reconfigure(spec types.VirtualMachineConfigSpec) error {
	debugf("vm.Reconfigure(%s)", vm.Name)
	task, err := vm.mo.Reconfigure(vm.vs.ctx, spec)
	if err != nil {
		return err
	}

	debugf("waiting for vm.Reconfigure %v", task)
	return task.Wait(vm.vs.ctx)
}

// func (vm *VirtualMachine) devices() (*deviceFinder, error) {
// 	devices, err := vm.mo.Device(vm.vs.ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &deviceFinder{devices}, nil
// }
