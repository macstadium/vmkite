package vsphere

import (
	"errors"
	"fmt"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// VirtualMachineCloneParams is passed to CloneVM
type VirtualMachineCloneParams struct {
	Name              string
	SrcName           string
	SrcSnapshot       string
	MemoryMB          int64
	NumCPUs           int32
	NumCoresPerSocket int32
	GuestInfo         map[string]string
}

func createCloneSpec(vm *VirtualMachine, params VirtualMachineCloneParams) (cs types.VirtualMachineCloneSpec, err error) {
	cs = types.VirtualMachineCloneSpec{
		Location: types.VirtualMachineRelocateSpec{
			DiskMoveType: "createNewChildDiskBacking",
		},
		PowerOn: false,
	}

	var o mo.VirtualMachine
	err = vm.mo.Properties(vm.vs.ctx, vm.mo.Reference(), []string{"snapshot"}, &o)
	if err != nil {
		return cs, err
	}

	if params.SrcSnapshot == "" {
		if o.Snapshot.CurrentSnapshot == nil {
			return cs, errors.New("Virtual machine doesn't have a current snapshot")
		}
		cs.Snapshot = o.Snapshot.CurrentSnapshot
	} else {
		snapshot, err := findSnapshotByName(o.Snapshot.RootSnapshotList, params.SrcSnapshot)
		if err != nil {
			return cs, err
		}
		cs.Snapshot = snapshot
	}

	return cs, nil
}

func createCloneReconfigureSpec(params VirtualMachineCloneParams) types.VirtualMachineConfigSpec {
	extraConfig := []types.BaseOptionValue{}

	for key, val := range params.GuestInfo {
		debugf("setting guestinfo.%s => %q", key, val)
		extraConfig = append(extraConfig,
			&types.OptionValue{Key: "guestinfo." + key, Value: val},
		)
	}

	return types.VirtualMachineConfigSpec{
		ExtraConfig:       extraConfig,
		MemoryMB:          params.MemoryMB,
		NumCPUs:           params.NumCPUs,
		NumCoresPerSocket: params.NumCoresPerSocket,
	}
}

func findSnapshotByName(tree []types.VirtualMachineSnapshotTree, snapshotName string) (*types.ManagedObjectReference, error) {
	for _, t := range tree {
		if t.Name == snapshotName {
			return &t.Snapshot, nil
		}
	}
	return nil, fmt.Errorf("Failed to find snapshot named %q", snapshotName)
}
