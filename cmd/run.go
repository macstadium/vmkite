package cmd

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/runner"
	"github.com/macstadium/vmkite/vsphere"
	metrics "github.com/rcrowley/go-metrics"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	buildkiteApiToken   string
	buildkiteAgentToken string
	buildkiteOrg        string
	buildkitePipelines  []string
	concurrency         int
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

	cmd.Flag("concurrency", "Limit how many concurrent jobs are run").
		Default("6").
		IntVar(&concurrency)

	addCreateVMFlags(cmd)

	cmd.Action(cmdRun)
}

func newMetricsRegistry() metrics.Registry {
	registry := metrics.NewRegistry()

	metrics.RegisterDebugGCStats(registry)
	go metrics.CaptureDebugGCStats(registry, 5*time.Second)

	metrics.RegisterRuntimeMemStats(registry)
	go metrics.CaptureRuntimeMemStats(registry, 5*time.Second)

	return registry
}

func cmdRun(c *kingpin.ParseContext) error {
	registry := newMetricsRegistry()
	go metrics.Log(registry, 30*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	vs, err := vsphere.NewSession(context.Background(), connectionParams)
	if err != nil {
		return err
	}

	bk, err := buildkite.NewSession(buildkiteOrg, buildkiteApiToken)
	if err != nil {
		return err
	}

	r := runner.NewRunner(vs, bk, runner.Params{
		Concurrency: concurrency,
		Pipelines:   buildkitePipelines,
	})

	return r.Run(vsphere.VirtualMachineCreationParams{
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
	})
}
