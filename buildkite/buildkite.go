package buildkite

import (
	"log"
	"strings"

	"gopkg.in/buildkite/go-buildkite.v2/buildkite"
)

type Session struct {
	ApiToken string
	Org      string
}

type VmkiteJob struct {
	ID       string
	Metadata VmkiteMetadata
}

func (bk *Session) VmkiteJobs() ([]VmkiteJob, error) {
	config, err := buildkite.NewTokenConfig(bk.ApiToken, false)
	if err != nil {
		return nil, err
	}
	client := buildkite.NewClient(config.Client())
	buildListOptions := buildkite.BuildsListOptions{
		State: []string{"scheduled", "running"},
	}
	debugf("Builds.ListByOrg(%s, ...)", bk.Org)
	builds, _, err := client.Builds.ListByOrg(bk.Org, &buildListOptions)
	if err != nil {
		return nil, err
	}
	jobs := make([]VmkiteJob, 0)
	for _, build := range builds {
		for _, job := range build.Jobs {
			metadata := parseAgentQueryRules(job.AgentQueryRules)
			if metadata.GuestID != "" && metadata.VMDK != "" {
				jobs = append(jobs, VmkiteJob{
					ID:       *job.ID,
					Metadata: metadata,
				})
			}
		}
	}
	return jobs, nil
}

type VmkiteMetadata struct {
	VMDK    string
	GuestID string
}

func parseAgentQueryRules(rules []string) VmkiteMetadata {
	debugf("parsing agent query rules: %#v", rules)

	metadata := VmkiteMetadata{}
	for _, r := range rules {
		parts := strings.SplitN(r, "=", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "vmkite-vmdk":
				metadata.VMDK = parts[1]
			case "vmkite-guestid":
				metadata.GuestID = parts[1]
			}
		}
	}
	return metadata
}

func debugf(format string, data ...interface{}) {
	log.Printf("[buildkite] "+format, data...)
}
