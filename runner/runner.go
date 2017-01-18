package runner

import (
	"log"

	"github.com/macstadium/vmkite/buildkite"
	"github.com/macstadium/vmkite/creator"
	"github.com/macstadium/vmkite/vsphere"
)

func RunOnce(vs *vsphere.Session, bk *buildkite.Session, params vsphere.VirtualMachineCreationParams) error {
	// TODO: long-running polling
	jobs, err := bk.VmkiteJobs()
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err := handleJob(j, vs, params); err != nil {
			return err
		}
	}
	return nil
}

func handleJob(job buildkite.VmkiteJob, vs *vsphere.Session, params vsphere.VirtualMachineCreationParams) (err error) {
	log.Printf("job %s => %s", job.ID, job.VMDK)
	params.SrcDiskPath = job.VMDK
	err = creator.CreateVM(vs, params)
	if err != nil {
		return err
	}

	return nil
}
