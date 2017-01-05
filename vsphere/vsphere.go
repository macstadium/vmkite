// Package vsphere is a vmkite-specific abstraction over vmware/govmomi,
// providing the ability to query and create vmkite macOS VMs using the VMware
// vSphere API.
package vsphere

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// ConnectionParams is passed by calling code to NewSession()
type ConnectionParams struct {
	Host     string
	User     string
	Pass     string
	Insecure bool
}

// HostSystem wraps govmomi's object.HostSystem
type HostSystem struct {
	mo *object.HostSystem

	ID string
	IP string
}

// Session holds state for a vSphere session;
// client connection, context, session-cached values like finder etc.
type Session struct {
	client     *govmomi.Client
	ctx        context.Context
	datacenter *object.Datacenter
	finder     *find.Finder
}

// VirtualMachineCreationParams is passed by calling code to Session.CreateVM()
type VirtualMachineCreationParams struct {
	BuildkiteAgentToken string
	DatastoreName       string
	HostSystem          HostSystem
	MacOsMinorVersion   int
	MemoryMB            int64
	Name                string
	NetworkLabel        string
	NumCPUs             int32
	NumCoresPerSocket   int32
	SrcDiskDataStore    string
	SrcDiskPath         string
}

// NewSession logs in to a new Session based on ConnectionParams
func NewSession(ctx context.Context, cp ConnectionParams) (*Session, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s/sdk", cp.Host))
	if err != nil {
		return nil, err
	}
	debugf("govmomi.NewClient(%s) user:%s", u.String(), cp.User)
	u.User = url.UserPassword(cp.User, cp.Pass)
	c, err := govmomi.NewClient(ctx, u, cp.Insecure)
	if err != nil {
		return nil, err
	}
	return &Session{
		ctx:    ctx,
		client: c,
	}, nil
}

// HostSystems finds vSphere hosts for the given path
func (vs *Session) HostSystems(path string) ([]HostSystem, error) {
	finder, err := vs.getFinder()
	if err != nil {
		return nil, err
	}
	debugf("finder.HostSystemList(%v)", path)
	results, err := finder.HostSystemList(vs.ctx, path)
	if err != nil {
		return nil, err
	}
	list := make([]HostSystem, 0)
	for _, hs := range results {
		list = append(list, HostSystem{
			mo: hs,
			IP: hs.Name(),
			ID: hs.Reference().Value,
		})
	}
	return list, nil
}

func (vs *Session) VirtualMachine(path string) (*VirtualMachine, error) {
	finder, err := vs.getFinder()
	if err != nil {
		return nil, err
	}
	debugf("finder.VirtualMachine(%v)", path)
	vm, err := finder.VirtualMachine(vs.ctx, path)
	if err != nil {
		return nil, err
	}
	return &VirtualMachine{
		vs:   vs,
		mo:   vm,
		Name: vm.Name(),
	}, nil
}

// VirtualMachines finds vSphere virtual machines for the given path.
func (vs *Session) VirtualMachines(path string) ([]VirtualMachine, error) {
	finder, err := vs.getFinder()
	if err != nil {
		return nil, err
	}
	debugf("finder.VirtualMachineList(%v)", path)
	results, err := finder.VirtualMachineList(vs.ctx, path)
	if err != nil {
		return nil, err
	}
	list := make([]VirtualMachine, 0)
	for _, vm := range results {
		debugf("vm.HostSystem() for %s", vm.Name())
		hs, err := vm.HostSystem(vs.ctx)
		if err != nil {
			return nil, err
		}
		list = append(list, VirtualMachine{
			vs:           vs,
			mo:           vm,
			Name:         vm.Name(),
			HostSystemID: hs.Reference().Value,
		})
	}
	return list, nil
}

// CreateVM launches a new macOS VM based on VirtualMachineCreationParams
func (vs *Session) CreateVM(params VirtualMachineCreationParams) (*VirtualMachine, error) {
	folder, err := vs.vmFolder()
	if err != nil {
		return nil, err
	}
	resourcePool, err := params.HostSystem.mo.ResourcePool(vs.ctx)
	if err != nil {
		return nil, err
	}
	configSpec, err := vs.createConfigSpec(params)
	if err != nil {
		return nil, err
	}
	debugf("CreateVM %s on %s (%s)", params.Name, params.HostSystem.ID, params.HostSystem.IP)
	task, err := folder.CreateVM(vs.ctx, configSpec, resourcePool, params.HostSystem.mo)
	if err != nil {
		return nil, err
	}
	debugf("waiting for CreateVM %v", task)
	if err := task.Wait(vs.ctx); err != nil {
		return nil, err
	}
	vm, err := vs.VirtualMachine(folder.InventoryPath + "/" + params.Name)
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (vs *Session) vmFolder() (*object.Folder, error) {
	if vs.datacenter == nil {
		return nil, errors.New("datacenter not loaded")
	}
	dcFolders, err := vs.datacenter.Folders(vs.ctx)
	if err != nil {
		return nil, err
	}
	return dcFolders.VmFolder, nil
}

func (vs *Session) createConfigSpec(params VirtualMachineCreationParams) (cs types.VirtualMachineConfigSpec, err error) {

	devices, err := addEthernet(nil, vs, params.NetworkLabel)
	if err != nil {
		return
	}

	devices, err = addSCSI(devices)
	if err != nil {
		return
	}

	devices, err = addDisk(devices, vs, params)
	if err != nil {
		return
	}

	devices, err = addUSB(devices)
	if err != nil {
		return
	}

	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return
	}

	extraConfig := []types.BaseOptionValue{
		&types.OptionValue{Key: "guestinfo.vmkite-buildkite-agent-token", Value: params.BuildkiteAgentToken},
		&types.OptionValue{Key: "guestinfo.vmkite-name", Value: params.Name},
		&types.OptionValue{Key: "guestinfo.vmkite-vmdk", Value: params.SrcDiskPath},
	}

	guestType, err := guestTypeForMinorVersion(params.MacOsMinorVersion)
	if err != nil {
		return
	}

	finder, err := vs.getFinder()
	if err != nil {
		return
	}
	debugf("finder.Datastore(%s)", params.DatastoreName)
	ds, err := finder.Datastore(vs.ctx, params.DatastoreName)
	if err != nil {
		return
	}
	fileInfo := &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", ds.Name()),
	}

	t := true
	cs = types.VirtualMachineConfigSpec{
		DeviceChange:        deviceChange,
		ExtraConfig:         extraConfig,
		Files:               fileInfo,
		GuestId:             guestType,
		MemoryMB:            params.MemoryMB,
		Name:                params.Name,
		NestedHVEnabled:     &t,
		NumCPUs:             params.NumCPUs,
		NumCoresPerSocket:   params.NumCoresPerSocket,
		VirtualICH7MPresent: &t,
		VirtualSMCPresent:   &t,
	}

	return
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
	eth := &types.VirtualE1000{
		types.VirtualEthernetCard{
			VirtualDevice: types.VirtualDevice{
				Key:     -1,
				Backing: backing,
			},
			AddressType: string(types.VirtualEthernetCardMacTypeGenerated),
		},
	}
	return append(devices, eth), nil
}

func addSCSI(devices object.VirtualDeviceList) (object.VirtualDeviceList, error) {
	scsi, err := object.SCSIControllerTypes().CreateSCSIController("scsi")
	if err != nil {
		return nil, err
	}
	return append(devices, scsi), nil
}

func addDisk(devices object.VirtualDeviceList, vs *Session, params VirtualMachineCreationParams) (object.VirtualDeviceList, error) {
	finder, err := vs.getFinder()
	if err != nil {
		return nil, err
	}
	debugf("finder.Datastore(%s)", params.SrcDiskDataStore)
	diskDatastore, err := finder.Datastore(vs.ctx, params.SrcDiskDataStore)
	if err != nil {
		return nil, err
	}
	controller, err := devices.FindDiskController("scsi")
	if err != nil {
		return nil, err
	}
	ds := diskDatastore.Reference()
	path := diskDatastore.Path(params.SrcDiskPath)
	disk := devices.CreateDisk(controller, ds, path)
	backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	backing.DiskMode = string(types.VirtualDiskModeIndependent_nonpersistent)
	disk = devices.ChildDisk(disk)
	return append(devices, disk), nil
}

func addUSB(devices object.VirtualDeviceList) (object.VirtualDeviceList, error) {
	t := true
	usb := &types.VirtualUSBController{AutoConnectDevices: &t, EhciEnabled: &t}
	return append(devices, usb), nil
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

func (vs *Session) getFinder() (*find.Finder, error) {
	if vs.finder == nil {
		debugf("find.NewFinder()")
		finder := find.NewFinder(vs.client.Client, true)
		debugf("finder.DefaultDatacenter()")
		dc, err := finder.DefaultDatacenter(vs.ctx)
		if err != nil {
			return nil, err
		}
		debugf("finder.SetDatacenter(%v)", dc)
		finder.SetDatacenter(dc)
		vs.datacenter = dc
		vs.finder = finder
	}
	return vs.finder, nil
}

func debugf(format string, data ...interface{}) {
	log.Printf("[vsphere] "+format, data...)
}
