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
	CreationParams vsphere.VirtualMachineCreationParams
	Pipelines      []string
}

func Run(vs *vsphere.Session, bk *buildkite.Session, params Params) error {
	st := &state{
		jobVMs: make(map[string]jobState),
	}
	for {
		jobs, err := bk.VmkiteJobs(buildkite.VmkiteJobQueryParams{
			Pipelines: params.Pipelines,
		})
		if err != nil {
			debugf("ERROR VmkiteJobs: %s", err)
			continue
		}
		for _, j := range jobs {
			if existing, ok := st.Get(j); ok {
				debugf("already created %s for job %s", existing.VmName, j.ID)
				continue
			}
			vmName, err := handleJob(j, vs, params.CreationParams)
			if err != nil {
				debugf("ERROR handleJob: %s", err)
				continue
			}
			debugf("created VM '%s' from '%s' for job %s", vmName, j.Metadata.VMDK, j.ID)
			st.Track(j, vmName, params.CreationParams.VirtualMachinePath)
		}
		if err = destroyFinished(st, vs, bk); err != nil {
			debugf("ERROR destroyFinished: %s", err)
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func RunOnce(vs *vsphere.Session, bk *buildkite.Session, params Params) error {
	jobs, err := bk.VmkiteJobs(buildkite.VmkiteJobQueryParams{
		Pipelines: params.Pipelines,
	})
	if err != nil {
		return err
	}
	for _, j := range jobs {
		vmName, err := handleJob(j, vs, params.CreationParams)
		if err != nil {
			return err
		}
		debugf("created VM '%s' from '%s' for job %s", vmName, j.Metadata.VMDK, j.ID)
	}
	return nil
}

func handleJob(job buildkite.VmkiteJob, vs *vsphere.Session, params vsphere.VirtualMachineCreationParams) (string, error) {
	debugf("handleJob(%s) => %s %s", job.String(), job.Metadata.VMDK, job.Metadata.GuestID)
	params.SrcDiskPath = job.Metadata.VMDK
	params.GuestID = job.Metadata.GuestID
	params.Name = job.VMName()

	vm, err := creator.CreateVM(vs, params)
	if err != nil {
		return "", err
	}
	return vm.Name, nil
}

func destroyFinished(s *state, vs *vsphere.Session, bk *buildkite.Session) error {
	for _, js := range s.List() {
		vm, err := vs.VirtualMachine(js.FullPath())
		if err != nil {
			return fmt.Errorf("Finding vm failed: %v", err)
		}
		poweredOn, err := vm.IsPoweredOn()
		if err != nil {
			return fmt.Errorf("vm.IsPoweredOn failed: %v", err)
		}
		if !poweredOn {
			debugf("destroying finished vm %s (%s since job created)", js.VmName, time.Now().Sub(js.CreatedAt))
			if err = vm.Destroy(true); err != nil {
				return err
			}
			s.Untrack(js.VmkiteJob)
		}
	}
	return nil
}

func debugf(format string, data ...interface{}) {
	log.Printf("[runner] "+format, data...)
}
