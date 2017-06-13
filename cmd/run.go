package cmd

import (
	"context"
	"log"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/runner"
	"github.com/macstadium/vmkite/vsphere"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var runParams = struct {
	buildkiteApiToken   string
	buildkiteAgentToken string
	buildkiteOrg        string
	buildkitePipelines  []string
	apiListenOn         string
	vmGuestInfo         map[string]string
}{
	vmGuestInfo: map[string]string{},
}

func ConfigureRun(app *kingpin.Application) {
	cmd := app.Command("run", "wait for Buildkite jobs, launch VMs")

	cmd.Flag("buildkite-agent-token", "Buildkite Agent Token").
		Required().
		StringVar(&runParams.buildkiteAgentToken)

	cmd.Flag("buildkite-api-token", "Buildkite API Token").
		Required().
		StringVar(&runParams.buildkiteApiToken)

	cmd.Flag("buildkite-org", "Buildkite organization slug").
		Required().
		StringVar(&runParams.buildkiteOrg)

	cmd.Flag("buildkite-pipeline", "Limit to a specific buildkite pipelines").
		StringsVar(&runParams.buildkitePipelines)

	cmd.Flag("api-listen", "The address and port for the api server to listen on").
		StringVar(&runParams.apiListenOn)

	cmd.Flag("vm-guest-info", "A set of key=value params to pass to the vm").
		StringMapVar(&cloneParams.vmGuestInfo)

	cmd.Action(cmdRun)
}

func cmdRun(c *kingpin.ParseContext) error {
	vs, err := vsphere.NewSession(context.Background(), globalParams.connectionParams)
	if err != nil {
		return err
	}

	bk, err := buildkite.NewSession(runParams.buildkiteOrg, runParams.buildkiteApiToken)
	if err != nil {
		return err
	}

	guestInfo := map[string]string{
		"vmkite-buildkite-agent-token": runParams.buildkiteAgentToken,
	}

	for k, v := range runParams.vmGuestInfo {
		guestInfo[k] = v
	}

	return runner.Run(vs, bk, runner.Params{
		Pipelines:   runParams.buildkitePipelines,
		ApiListenOn: runParams.apiListenOn,
		GuestInfo:   guestInfo,
	})
}

func debugf(format string, data ...interface{}) {
	log.Printf("[cmd] "+format, data...)
}
