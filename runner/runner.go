package runner

import (
	"log"
	"time"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"
)

type state struct {
	jobVMs map[string]string
}

func Run(vs *vsphere.Session, bk *buildkite.Session, params vsphere.VirtualMachineCreationParams) error {
	st := state{
		jobVMs: make(map[string]string),
	}
	for {
		jobs, err := bk.VmkiteJobs()
		if err != nil {
			debugf("ERROR VmkiteJobs: %s", err)
			continue
		}
		for _, j := range jobs {
			if existingVmName, ok := st.jobVMs[j.ID]; ok {
				debugf("already created %s for job %s", existingVmName, j.ID)
				continue
			}
			vmName, err := handleJob(j, vs, params)
			if err != nil {
				debugf("ERROR handleJob: %s", err)
				continue
			}
			debugf("created VM '%s' from '%s' for job %s", vmName, j.Metadata.VMDK, j.ID)
			st.jobVMs[j.ID] = vmName
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func RunOnce(vs *vsphere.Session, bk *buildkite.Session, params vsphere.VirtualMachineCreationParams) error {
	jobs, err := bk.VmkiteJobs()
	if err != nil {
		return err
	}
	for _, j := range jobs {
		vmName, err := handleJob(j, vs, params)
		if err != nil {
			return err
		}
		debugf("created VM '%s' from '%s' for job %s", vmName, j.Metadata.VMDK, j.ID)
	}
	return nil
}

func handleJob(job buildkite.VmkiteJob, vs *vsphere.Session, params vsphere.VirtualMachineCreationParams) (string, error) {
	debugf("job %s => %s %s", job.ID, job.Metadata.VMDK, job.Metadata.GuestID)
	params.SrcDiskPath = job.Metadata.VMDK
	params.GuestID = job.Metadata.GuestID

	debugf("vm params %#v", params)
	vm, err := creator.CreateVM(vs, params)
	if err != nil {
		return "", err
	}
	return vm.Name, nil
}

func debugf(format string, data ...interface{}) {
	log.Printf("[runner] "+format, data...)
}
