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
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/session"
)

const keepAliveDuration = time.Second * 30

// ConnectionParams is passed by calling code to NewSession()
type ConnectionParams struct {
	Host     string
	User     string
	Pass     string
	Insecure bool
}

// Session holds state for a vSphere session;
// client connection, context, session-cached values like finder etc.
type Session struct {
	client     *govmomi.Client
	ctx        context.Context
	datacenter *object.Datacenter
	finder     *find.Finder
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
	// prevent vsphere session from dropping
	c.Client.RoundTripper = session.KeepAlive(
		c.Client.RoundTripper,
		keepAliveDuration,
	)
	return &Session{
		ctx:    ctx,
		client: c,
	}, nil
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

// CreateVM launches a new VM based on VirtualMachineCreationParams
func (vs *Session) CreateVM(params VirtualMachineCreationParams) (*VirtualMachine, error) {
	// finder, err := vs.getFinder()
	// if err != nil {
	// 	return nil, err
	// }
	// folder, err := vs.vmFolder()
	// if err != nil {
	// 	return nil, err
	// }
	// debugf("finder.ClusterComputeResource(%s)", params.ClusterPath)
	// cluster, err := finder.ClusterComputeResource(vs.ctx, params.ClusterPath)
	// if err != nil {
	// 	return nil, err
	// }
	// debugf("cluster.ResourcePool()")
	// resourcePool, err := cluster.ResourcePool(vs.ctx)
	// if err != nil {
	// 	return nil, err
	// }
	configSpec, err := createConfigSpec(vs, params)
	if err != nil {
		return nil, err
	}

	spew.Dump(configSpec)
	// }
	// debugf("folder.CreateVM %s on %s", params.Name, resourcePool)
	// task, err := folder.CreateVM(vs.ctx, configSpec, resourcePool, nil)
	// if err != nil {
	// 	return nil, err
	// }
	// debugf("waiting for CreateVM %v", task)
	// if err := task.Wait(vs.ctx); err != nil {
	// 	return nil, err
	// }
	// vm, err := vs.VirtualMachine(folder.InventoryPath + "/" + params.Name)
	// if err != nil {
	// 	return nil, err
	// }
	// return vm, nil

	return nil, errors.New("Not implemented")
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
