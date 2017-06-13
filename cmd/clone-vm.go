package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var cloneParams = struct {
	buildkiteAgentToken string
	vmSourceName        string
	vmSourceSnapshot    string
	vmGuestInfo         map[string]string
}{
	vmGuestInfo: map[string]string{},
}

func ConfigureCloneVM(app *kingpin.Application) {
	cmd := app.Command("clone-vm", "clone a virtual machine or template")

	cmd.Flag("buildkite-agent-token", "Buildkite Agent Token").
		Required().
		StringVar(&cloneParams.buildkiteAgentToken)

	cmd.Flag("vm-source-snapshot", "an optional snapshot to use for the clone").
		Required().
		StringVar(&cloneParams.vmSourceSnapshot)

	cmd.Flag("vm-source-name", "The source VM").
		StringVar(&cloneParams.vmSourceName)

	cmd.Flag("vm-guest-info", "A set of key=value params to pass to the vm").
		StringMapVar(&cloneParams.vmGuestInfo)

	cmd.Action(cmdCloneVM)
}

func cmdCloneVM(c *kingpin.ParseContext) error {
	ctx := context.Background()

	vs, err := vsphere.NewSession(ctx, globalParams.connectionParams)
	if err != nil {
		return err
	}

	guestInfo := map[string]string{
		"vmkite-buildkite-agent-token": cloneParams.buildkiteAgentToken,
	}

	for k, v := range cloneParams.vmGuestInfo {
		guestInfo[k] = v
	}

	_, err = creator.CloneVM(vs, vsphere.VirtualMachineCloneParams{
		SrcName:     cloneParams.vmSourceName,
		SrcSnapshot: cloneParams.vmSourceSnapshot,
		Name:        fmt.Sprintf("vmkite-%s", time.Now().Format("200612-150405")),
		GuestInfo:   guestInfo,
	})
	if err != nil {
		return err
	}

	return nil
}
