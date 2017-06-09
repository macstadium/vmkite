package cmd

import (
	"context"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/runner"
	"github.com/macstadium/vmkite/vsphere"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	buildkiteApiToken   string
	buildkiteAgentToken string
	buildkiteOrg        string
	buildkitePipelines  []string
	runOnce             bool
)

func ConfigureRun(app *kingpin.Application) {
	cmd := app.Command("run", "wait for Buildkite jobs, launch VMs")

	cmd.Flag("buildkite-agent-token", "Buildkite Agent Token").
		Required().
		StringVar(&buildkiteAgentToken)

	cmd.Flag("buildkite-api-token", "Buildkite API Token").
		Required().
		StringVar(&buildkiteApiToken)

	cmd.Flag("buildkite-org", "Buildkite organization slug").
		Required().
		StringVar(&buildkiteOrg)

	cmd.Flag("buildkite-pipeline", "Limit to a specific buildkite pipelines").
		StringsVar(&buildkitePipelines)

	cmd.Flag("once", "Run once, launch for waiting jobs, exit").
		BoolVar(&runOnce)

	addCreateVMFlags(cmd)

	cmd.Action(cmdRun)
}

func cmdRun(c *kingpin.ParseContext) error {
	vs, err := vsphere.NewSession(context.Background(), connectionParams)
	if err != nil {
		return err
	}

	bk, err := buildkite.NewSession(buildkiteOrg, buildkiteApiToken)
	if err != nil {
		return err
	}

	params := runner.Params{
		Pipelines: buildkitePipelines,
		CreationParams: vsphere.VirtualMachineCreationParams{
			BuildkiteAgentToken: buildkiteAgentToken,
			ClusterPath:         vmClusterPath,
			VirtualMachinePath:  vmPath,
			DatastoreName:       vmDS,
			MemoryMB:            vmMemoryMB,
			Name:                "", // automatic
			NetworkLabel:        vmNetwork,
			NumCPUs:             vmNumCPUs,
			NumCoresPerSocket:   vmNumCoresPerSocket,
			SrcDiskDataStore:    vmdkDS,
			SrcDiskPath:         "", // per-job
			GuestInfo:           vmGuestInfo,
		},
	}

	if runOnce {
		return runner.RunOnce(vs, bk, params)
	} else {
		return runner.Run(vs, bk, params)
	}
}
