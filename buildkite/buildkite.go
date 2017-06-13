package buildkite

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gopkg.in/buildkite/go-buildkite.v2/buildkite"
)

type Session struct {
	Org    string
	client *buildkite.Client
}

func NewSession(org string, apiToken string) (*Session, error) {
	config, err := buildkite.NewTokenConfig(apiToken, false)
	if err != nil {
		return nil, err
	}
	return &Session{
		Org:    org,
		client: buildkite.NewClient(config.Client()),
	}, nil
}

type VmkiteJob struct {
	ID          string
	BuildNumber string
	Pipeline    string
	CreatedAt   time.Time
	Metadata    VmkiteMetadata
}

func (v *VmkiteJob) String() string {
	return fmt.Sprintf("%s/%s/%s", v.Pipeline, v.BuildNumber, v.ID)
}

func (v VmkiteJob) VMName() string {
	return fmt.Sprintf("%s-%s-%s-%s",
		v.Metadata.Template, v.Pipeline, v.BuildNumber, time.Now().Format("200612-150405"))
}

type VmkiteJobQueryParams struct {
	Pipelines []string
}

func (bk *Session) VmkiteJobs(query VmkiteJobQueryParams) ([]VmkiteJob, error) {
	if len(query.Pipelines) > 0 {
		jobs := make([]VmkiteJob, 0)
		for _, pipeline := range query.Pipelines {
			debugf("Builds.ListByPipeline(%s, %s, ...)", bk.Org, pipeline)
			builds, _, err := bk.client.Builds.ListByPipeline(bk.Org, pipeline, &buildkite.BuildsListOptions{
				State: []string{"scheduled", "running"},
			})
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, readJobsFromBuilds(builds)...)
		}
		return jobs, nil
	}

	debugf("Builds.ListByOrg(%s, ...)", bk.Org)
	builds, _, err := bk.client.Builds.ListByOrg(bk.Org, &buildkite.BuildsListOptions{
		State: []string{"scheduled", "running"},
	})
	if err != nil {
		return nil, err
	}

	return readJobsFromBuilds(builds), nil
}

func readJobsFromBuilds(builds []buildkite.Build) []VmkiteJob {
	jobs := make([]VmkiteJob, 0)
	for _, build := range builds {
		for _, job := range build.Jobs {
			metadata := parseAgentQueryRules(job.AgentQueryRules)
			if metadata.Template != "" {
				jobs = append(jobs, VmkiteJob{
					ID:          *job.ID,
					BuildNumber: strconv.Itoa(*build.Number),
					Pipeline:    *build.Pipeline.Slug,
					Metadata:    metadata,
					CreatedAt:   job.CreatedAt.Time,
				})
			}
		}
	}
	return jobs
}

func (bk *Session) IsFinished(job VmkiteJob) (bool, error) {
	debugf("Builds.Get(%s, %s, %s)", bk.Org, job.Pipeline, job.BuildNumber)
	build, _, err := bk.client.Builds.Get(bk.Org, job.Pipeline, job.BuildNumber)
	if err != nil {
		return false, err
	}
	for _, buildJob := range build.Jobs {
		if *buildJob.ID == job.ID {
			switch *buildJob.State {
			case "scheduled", "running":
				return false, nil
			}
			return true, nil
		}
	}
	return false, nil
}

type VmkiteMetadata struct {
	Template string
	Queue    string
}

func parseAgentQueryRules(rules []string) VmkiteMetadata {
	metadata := VmkiteMetadata{}
	for _, r := range rules {
		parts := strings.SplitN(r, "=", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "vmkite-template":
				metadata.Template = parts[1]
			case "queue":
				metadata.Queue = parts[1]
			}
		}
	}
	return metadata
}

func debugf(format string, data ...interface{}) {
	log.Printf("[buildkite] "+format, data...)
}
