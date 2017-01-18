package cmd

import (
	"context"
	"log"

	"github.com/lox/vmkite/buildkite"
	"github.com/lox/vmkite/creator"
	"github.com/lox/vmkite/vsphere"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	buildkiteApiToken   string
	buildkiteAgentToken string
	buildkiteOrg        string
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

	addCreateVMFlags(cmd)

	cmd.Action(cmdRun)
}

func cmdRun(c *kingpin.ParseContext) error {
	vs, err := vsphere.NewSession(context.Background(), connectionParams)
	if err != nil {
		return err
	}
	// TODO: long-running polling
	jobs, err := buildkite.VmkiteJobs(
		buildkiteApiToken,
		buildkiteOrg,
	)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err := handleJob(j, vs); err != nil {
			return err
		}
	}
	return nil
}

func handleJob(job buildkite.VmkiteJob, vs *vsphere.Session) (err error) {
	log.Printf("job %s => %s", job.ID, job.VMDK)

	params := vsphere.VirtualMachineCreationParams{
		BuildkiteAgentToken: buildkiteAgentToken,
		ClusterPath:         vmClusterPath,
		DatastoreName:       vmDS,
		MacOsMinorVersion:   macOsMinor,
		MemoryMB:            vmMemoryMB,
		Name:                "",
		NetworkLabel:        vmNetwork,
		NumCPUs:             vmNumCPUs,
		NumCoresPerSocket:   vmNumCoresPerSocket,
		SrcDiskDataStore:    vmdkDS,
		SrcDiskPath:         job.VMDK,
	}

	err = creator.CreateVM(vs, params)
	if err != nil {
		return err
	}

	return nil
}
