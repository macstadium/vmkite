package creator

import (
	"fmt"
	"time"

	"github.com/lox/vmkite/vsphere"
)

func CreateVM(vs *vsphere.Session, params vsphere.VirtualMachineCreationParams) error {
	if params.Name == "" {
		ts := time.Now().Format("200612-150405")
		params.Name = fmt.Sprintf("vmkite-host-macOS_10_%d-%s", params.MacOsMinorVersion, ts)
	}
	vm, err := vs.CreateVM(params)
	if err != nil {
		return err
	}
	if err := vm.PowerOn(); err != nil {
		return err
	}
	return nil

}
