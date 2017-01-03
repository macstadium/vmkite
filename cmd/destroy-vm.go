package cmd

import (
	"context"

	"github.com/lox/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	vmNames []string
)

func ConfigureDestroyVM(app *kingpin.Application) {
	cmd := app.Command("destroy-vm", "destroy a virtual machine")

	cmd.Arg("name", "name of virtual machine to destroy").
		Required().
		StringsVar(&vmNames)

	cmd.Action(cmdDestroyVM)
}

func cmdDestroyVM(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, connectionParams)
	if err != nil {
		return err
	}

	for _, vmName := range vmNames {
		vm, err := vs.VirtualMachine(vmPath + "/" + vmName)
		if err != nil {
			return err
		}
		if err = vm.Destroy(true); err != nil {
			return err
		}
	}

	return nil
}
