package vsphere

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

type VirtualMachineCreationParams struct {
	Name              string
	SrcName           string
	MemoryMB          int64
	NumCPUs           int32
	NumCoresPerSocket int32
	Network           string
	Datastore         string
	GuestInfo         map[string]string
	GuestID           string
	ClusterPath       string
	ResourcePath      string
}

func createConfigSpec(vs *Session, params VirtualMachineCreationParams) (*types.VirtualMachineConfigSpec, error) {
	templateVM, err := vs.VirtualMachine(params.SrcName)
	if err != nil {
		return nil, err
	}

	templateDevices, err := templateVM.mo.Device(vs.ctx)
	if err != nil {
		return nil, err
	}

	for _, device := range templateDevices {
		if card, ok := device.(types.BaseVirtualEthernetCard); ok {
			if backing, ok := card.GetVirtualEthernetCard().VirtualDevice.Backing.(*types.VirtualEthernetCardNetworkBackingInfo); ok {
				if params.Network == "" {
					params.Network = backing.DeviceName
				}

			}
		} else if disk, ok := device.(*types.VirtualDisk); ok {
			if backing, ok := disk.VirtualDevice.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
				spew.Dump(backing.FileName)
			}

		}
	}

	devices, err := addEthernet(nil, vs, params.Network)
	if err != nil {
		return nil, err
	}

	// devices, err = addSCSI(devices)
	// if err != nil {
	// 	return
	// }

	// devices, err = addDisk(devices, vs, params)
	// if err != nil {
	// 	return
	// }

	// devices, err = addUSB(devices)
	// if err != nil {
	// 	return
	// }

	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}

	extraConfig := []types.BaseOptionValue{}

	if params.GuestInfo != nil {
		for key, val := range params.GuestInfo {
			debugf("setting guestinfo.%s => %q", key, val)
			extraConfig = append(extraConfig,
				&types.OptionValue{Key: "guestinfo." + key, Value: val},
			)
		}
	}

	// ensure a consistent pci slot for the ethernet card, helps systemd
	extraConfig = append(extraConfig,
		&types.OptionValue{Key: "ethernet0.pciSlotNumber", Value: "32"},
	)

	fileInfo := &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", params.Datastore),
	}

	t := true
	cs := &types.VirtualMachineConfigSpec{
		DeviceChange:        deviceChange,
		ExtraConfig:         extraConfig,
		Files:               fileInfo,
		GuestId:             params.GuestID,
		MemoryMB:            params.MemoryMB,
		Name:                params.Name,
		NestedHVEnabled:     &t,
		NumCPUs:             params.NumCPUs,
		NumCoresPerSocket:   params.NumCoresPerSocket,
		VirtualICH7MPresent: &t,
		VirtualSMCPresent:   &t,
	}

	return cs, err
}

func addEthernet(devices object.VirtualDeviceList, vs *Session, label string) (object.VirtualDeviceList, error) {
	finder, err := vs.getFinder()
	if err != nil {
		return nil, err
	}

	path := "*" + label
	debugf("finder.Network(%s)", path)
	network, err := finder.Network(vs.ctx, path)
	if err != nil {
		return nil, err
	}

	backing, err := network.EthernetCardBackingInfo(vs.ctx)
	if err != nil {
		return nil, err
	}

	device, err := object.EthernetCardTypes().CreateEthernetCard("e1000e", backing)
	if err != nil {
		return nil, err
	}

	card := device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
	card.AddressType = string(types.VirtualEthernetCardMacTypeGenerated)

	return append(devices, device), nil
}

// func addSCSI(devices object.VirtualDeviceList) (object.VirtualDeviceList, error) {
// 	scsi, err := object.SCSIControllerTypes().CreateSCSIController("scsi")
// 	if err != nil {
// 		return nil, err
// 	}
// 	return append(devices, scsi), nil
// }

// func addDisk(devices object.VirtualDeviceList, vs *Session, params VirtualMachineCreationParams) (object.VirtualDeviceList, error) {
// 	finder, err := vs.getFinder()
// 	if err != nil {
// 		return nil, err
// 	}

// 	debugf("finder.Datastore(%s)", params.Datastore)
// 	diskDatastore, err := finder.Datastore(vs.ctx, params.Datastore)
// 	if err != nil {
// 		return nil, err
// 	}

// 	controller, err := devices.FindDiskController("scsi")
// 	if err != nil {
// 		return nil, err
// 	}

// 	disk := devices.CreateDisk(
// 		controller,
// 		diskDatastore.Reference(),
// 		diskDatastore.Path("blah"),
// 	)

// 	backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
// 	backing.ThinProvisioned = types.NewBool(true)
// 	backing.DiskMode = string(types.VirtualDiskModeIndependent_nonpersistent)

// 	return append(devices, disk), nil
// }

// func addUSB(devices object.VirtualDeviceList) (object.VirtualDeviceList, error) {
// 	t := true
// 	usb := &types.VirtualUSBController{AutoConnectDevices: &t, EhciEnabled: &t}
// 	return append(devices, usb), nil
// }
