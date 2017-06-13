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
	ApiListenOn string
	GuestInfo   map[string]string
	ClusterPath string
}

func Run(vs *vsphere.Session, bk *buildkite.Session, params Params) error {
	st := newState()

	// startup an api listener to track granular state changes in vm
	api, err := apiListen(params.ApiListenOn, buildHookHandler(st, vs))
	if err != nil {
		return err
	}

	runner := &jobRunner{
		vs:        vs,
		state:     st,
		api:       api,
		guestInfo: params.GuestInfo,
	}

	for {
		jobs, err := bk.VmkiteJobs(buildkite.VmkiteJobQueryParams{
			Pipelines: params.Pipelines,
		})
		if err != nil {
			debugf("ERROR VmkiteJobs: %v", err)
			continue
		}

		// spawn VMs for all pending jobs
		for _, job := range jobs {
			if existing, ok := st.GetJob(job.ID); ok {
				debugf("Ignoring existing %s", existing.VmName)
				continue
			}

			vmName, err := runner.Run(job)
			if err != nil {
				debugf("ERROR jobRunner.Run() %v", err)
				continue
			}

			st.TrackJob(job, vmName)
		}

		// if err = destroyAllFinished(st, vs, bk); err != nil {
		// 	debugf("ERROR destroyFinished: %s", err)
		// }
		time.Sleep(2 * time.Second)
	}

	return nil
}

type jobRunner struct {
	job       buildkite.VmkiteJob
	api       *api
	vs        *vsphere.Session
	guestInfo map[string]string
	state     *state
}

func (jr *jobRunner) Run(job buildkite.VmkiteJob) (string, error) {
	token, err := jr.api.RegisterJob(job.ID)
	if err != nil {
		return "", err
	}

	guestInfo := jr.guestInfo
	guestInfo["vmkite-api"] = jr.api.Addr().String()
	guestInfo["vmkite-api-token"] = token
	guestInfo["vmkite-template"] = job.Metadata.Template
	guestInfo["vmkite-name"] = job.VMName()

	cloneParams := vsphere.VirtualMachineCloneParams{
		Name:      guestInfo["vmkite-name"],
		SrcName:   job.Metadata.Template,
		GuestInfo: guestInfo,
	}

	snapshot, err := jr.state.FindBestSnapshot(job)
	if err != nil {
		debugf("ERROR FindSnapshot: %v", err)
	} else {
		debugf("Using snapshot: %v", snapshot)
		cloneParams.SrcName = snapshot.VMName
		cloneParams.SrcSnapshot = snapshot.Snapshot
	}

	vm, err := creator.CloneVM(jr.vs, cloneParams)
	if err != nil {
		return "", err
	}

	return vm.Name, nil
}

func buildHookHandler(st *state, vs *vsphere.Session) apiHookHandler {
	return apiHookHandler(func(hook string, jobID string) error {
		js, ok := st.GetJob(jobID)
		if !ok {
			return fmt.Errorf("Finding find a VM for: %v", jobID)
		}

		debugf("Hook %s for %s is %v after job start time",
			hook, js.String(), time.Now().Sub(js.CreatedAt))

		switch hook {
		case "pre-command":
			debugf("handling snapshot on pre-command hook for %s", jobID)

			vm, err := vs.VirtualMachine(js.VmName)
			if err != nil {
				return fmt.Errorf("Finding vm failed: %v", err)
			}

			descr := fmt.Sprintf(
				"Triggered by %s on %s",
				hook, js.String(),
			)

			go func() {
				err = vm.CreateSnapshot(hook, descr, false, false)
				if err != nil {
					debugf("Error creating snapshot: %v", err)
					return
				}

				st.TrackSnapshot(js.VmkiteJob, js.VmName, hook)
			}()
		}

		return nil
	})
}

func destroyAllFinished(s *state, vs *vsphere.Session, bk *buildkite.Session) error {
	for _, js := range s.ListJobs() {
		vm, err := vs.VirtualMachine(js.VmName)
		if err != nil {
			return fmt.Errorf("Finding vm failed: %v", err)
		}

		poweredOn, err := vm.IsPoweredOn()
		if err != nil {
			return fmt.Errorf("vm.IsPoweredOn failed: %v", err)
		}

		if !poweredOn {
			debugf("destroying finished vm %s", js.VmName)
			if err = vm.Destroy(true); err != nil {
				return err
			}
			s.UntrackJob(js.VmkiteJob)
		}
	}
	return nil
}

func debugf(format string, data ...interface{}) {
	log.Printf("[runner] "+format, data...)
}
