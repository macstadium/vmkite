package runner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"
)

type Params struct {
	Pipelines      []string
	Concurrency    int
	ApiListenOn    string
	ApiTokenSecret string
}

type Runner struct {
	vs     *vsphere.Session
	bk     *buildkite.Session
	params Params
}

func NewRunner(vs *vsphere.Session, bk *buildkite.Session, p Params) *Runner {
	return &Runner{
		vs:     vs,
		bk:     bk,
		params: p,
	}
}

func (r *Runner) Run(createParams vsphere.VirtualMachineCreationParams) error {
	var wg sync.WaitGroup

	api, err := newApiListener(r.params.ApiListenOn, r.params.ApiTokenSecret)
	if err != nil {
		return err
	}

	jobs := r.bk.PollJobs(buildkite.VmkiteJobQueryParams{
		Pipelines: r.params.Pipelines,
	})

	for i := 0; i < r.params.Concurrency; i++ {
		debugf("spawning runner %d", i+1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				token, ch, err := api.Subscribe(job)
				if err != nil {
					debugf("Error subscribing to hook events: %v", err)
					continue
				}

				createParams.GuestInfo["vmkite-api"] = api.Addr().String()
				createParams.GuestInfo["vmkite-api-token"] = token

				if err := r.runJob(createParams, job, ch); err != nil {
					debugf("Error running job: %v", err)
				}

				api.Release(job)
			}
		}()
	}

	wg.Wait()
	return nil
}

func (r *Runner) runJob(createParams vsphere.VirtualMachineCreationParams, job buildkite.VmkiteJob, events chan apiHookEvent) error {
	debugf("running job %v", job.ID)
	vm, err := r.createVMForJob(createParams, job)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	debugf("waiting for job %v to finish", job.ID)
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case event := <-events:
			debugf("read event %s from job %s (%v after job created)",
				event.Event, event.JobID, event.Timestamp.Sub(job.CreatedAt))

		case <-ticker.C:
			poweredOn, err := vm.IsPoweredOn()
			if err != nil {
				return fmt.Errorf("vm.IsPoweredOn failed: %v", err)
			}

			if !poweredOn {
				debugf("VM is powered off, destroying")
				return vm.Destroy(true)
			}

		case <-ctx.Done():
			return errors.New("Timed out waiting for VM power-off")
		}
	}
}

func (r *Runner) createVMForJob(createParams vsphere.VirtualMachineCreationParams, job buildkite.VmkiteJob) (*vsphere.VirtualMachine, error) {
	if existing, err := r.vs.VirtualMachine(job.VMName()); err == nil {
		debugf("vm %s already exists, skipping create", existing.Name)
		return existing, nil
	}

	// add parameters from the job
	createParams.SrcDiskPath = job.Metadata.VMDK
	createParams.GuestID = job.Metadata.GuestID
	createParams.Name = job.VMName()

	debugf("createVM(%s) => %s %s", job.String(), job.Metadata.VMDK, job.Metadata.GuestID)
	vm, err := creator.CreateVM(r.vs, createParams)
	if err != nil {
		return nil, err
	}

	debugf("created VM %q for job %s", vm.Name, job.String())
	return vm, nil
}

func debugf(format string, data ...interface{}) {
	log.Printf("[runner] "+format, data...)
}
