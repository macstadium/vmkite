package cmd

import (
	"context"

	"github.com/lox/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	vmName string
)

func ConfigureDestroyVM(app *kingpin.Application) {
	cmd := app.Command("destroy-vm", "destroy a virtual machine")

	cmd.Arg("name", "name of virtual machine to destroy").
		Required().
		StringVar(&vmName)

	cmd.Action(cmdDestroyVM)
}

func cmdDestroyVM(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, vsphere.ConnectionParams{
		Host:     vsHost,
		User:     vsUser,
		Pass:     vsPass,
		Insecure: vsInsecure,
	})

	vm, err := vs.VirtualMachine(vmPath + "/" + vmName)
	if err != nil {
		return err
	}

	if err = vm.Destroy(true); err != nil {
		return err
	}

	return nil
}
