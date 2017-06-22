package runner

import (
	"fmt"
	"log"
	"time"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"
)

type Params struct {
	Pipelines   []string
	Concurrency int
}

type Runner struct {
	vs     *vsphere.Session
	bk     *buildkite.Session
	state  *state
	params Params
}

func NewRunner(vs *vsphere.Session, bk *buildkite.Session, p Params) *Runner {
	return &Runner{
		vs:     vs,
		bk:     bk,
		params: p,
		state: &state{
			jobVMs: make(map[string]jobState),
		},
	}
}

func (r *Runner) Run(createParams vsphere.VirtualMachineCreationParams) error {
	for {
		jobs, err := r.bk.ListJobs(buildkite.VmkiteJobQueryParams{
			Pipelines: r.params.Pipelines,
		})
		if err != nil {
			debugf("ERROR VmkiteJobs: %v", err)
			continue
		}

		for _, job := range jobs {
			if r.isJobAlreadyCreated(job) {
				continue
			}

			if r.atConcurrencyLimit() {
				debugf("hit concurrency limit of %d, waiting to launch vm", r.params.Concurrency)
				continue
			}

			// add parameters from the job
			createParams.SrcDiskPath = job.Metadata.VMDK
			createParams.GuestID = job.Metadata.GuestID
			createParams.Name = job.VMName()

			debugf("createVM(%s) => %s %s", job.String(), job.Metadata.VMDK, job.Metadata.GuestID)
			vm, err := creator.CreateVM(r.vs, createParams)
			if err != nil {
				debugf("ERROR createVM: %#v", err)
				continue
			}

			debugf("created VM %q for job %s", vm.Name, job.String())
			r.state.Track(job, vm.Name)
		}

		if err = r.destroyAllFinished(); err != nil {
			debugf("ERROR destroyAllFinished: %s", err)
		}

		time.Sleep(2 * time.Second)
	}
}

func (r *Runner) atConcurrencyLimit() bool {
	return r.params.Concurrency > 0 && r.state.Len() >= r.params.Concurrency
}

func (r *Runner) isJobAlreadyCreated(job buildkite.VmkiteJob) bool {
	if existing, ok := r.state.Get(job); ok {
		debugf("vm %s was created %v ago", existing.VmName, time.Now().Sub(existing.CreatedAt))
		return true
	}

	if existing, err := r.vs.VirtualMachine(job.VMName()); err == nil {
		debugf("vm %s already exists, but isn't tracked yet", existing.Name)
		r.state.Track(job, existing.Name)
		return true
	}

	return false
}

func (r *Runner) destroyAllFinished() error {
	for _, jobState := range r.state.List() {
		vm, err := r.vs.VirtualMachine(jobState.FullPath())
		if err != nil {
			return fmt.Errorf("Finding vm failed: %v", err)
		}

		poweredOn, err := vm.IsPoweredOn()
		if err != nil {
			return fmt.Errorf("vm.IsPoweredOn failed: %v", err)
		}

		if !poweredOn {
			debugf("destroying finished vm %s (%s since job created)",
				jobState.VmName, time.Now().Sub(jobState.CreatedAt))

			if err = vm.Destroy(true); err != nil {
				return err
			}

			r.state.Untrack(jobState.VmkiteJob)
		}
	}
	return nil
}

func debugf(format string, data ...interface{}) {
	log.Printf("[runner] "+format, data...)
}
