package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var createParams = struct {
	buildkiteAgentToken string
	vmSourceName        string
	vmNetwork           string
	vmDatasource        string
	vmGuestInfo         map[string]string
}{
	vmGuestInfo: map[string]string{},
}

func ConfigureCreateVM(app *kingpin.Application) {
	cmd := app.Command("create-vm", "create a virtual machine")

	cmd.Flag("vm-source-name", "A vm to use as a template for the vm").
		StringVar(&createParams.vmSourceName)

	cmd.Flag("buildkite-agent-token", "Buildkite Agent Token").
		Required().
		StringVar(&createParams.buildkiteAgentToken)

	cmd.Flag("vm-guest-info", "A set of key=value params to pass to the vm").
		StringMapVar(&createParams.vmGuestInfo)

	cmd.Flag("vm-network", "The network to use for the vm").
		StringVar(&createParams.vmNetwork)

	cmd.Flag("vm-datastore", "The datastore to use for the vm").
		StringVar(&createParams.vmDatasource)

	cmd.Action(cmdCreateVM)
}

func cmdCreateVM(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, globalParams.connectionParams)
	if err != nil {
		return err
	}

	guestInfo := map[string]string{
		"vmkite-buildkite-agent-token": createParams.buildkiteAgentToken,
	}

	for k, v := range createParams.vmGuestInfo {
		guestInfo[k] = v
	}

	_, err = creator.CreateVM(vs, vsphere.VirtualMachineCreationParams{
		SrcName:   createParams.vmSourceName,
		Name:      fmt.Sprintf("vmkite-%s", time.Now().Format("200612-150405")),
		Network:   createParams.vmNetwork,
		Datastore: createParams.vmDatasource,
		GuestInfo: guestInfo,
	})

	return err
}
