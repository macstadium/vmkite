package vsphere

import "github.com/vmware/govmomi/object"

// VirtualMachine wraps govmomi's object.VirtualMachine
type VirtualMachine struct {
	vs *Session

	mo *object.VirtualMachine

	HostSystemID string
	Name         string
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

	debugf("Destroy %s", vm.Name)
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
	debugf("PowerOff %s", vm.Name)
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
	debugf("PowerOn %s", vm.Name)
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
