package runner

import (
	"errors"
	"sync"

	"github.com/macstadium/vmkite/buildkite"
)

type snapshot struct {
	VMName   string
	Snapshot string
}

type jobState struct {
	buildkite.VmkiteJob
	VmName string
}

type state struct {
	jobVMs    map[string]jobState
	snapshots map[string][]snapshot
	sync.RWMutex
}

func newState() *state {
	return &state{
		jobVMs:    make(map[string]jobState),
		snapshots: make(map[string][]snapshot),
	}
}

func (st *state) ListJobs() []jobState {
	st.RLock()
	defer st.RUnlock()
	states := []jobState{}
	for _, state := range st.jobVMs {
		states = append(states, state)
	}
	return states
}

func (st *state) GetJob(jobID string) (jobState, bool) {
	st.RLock()
	defer st.RUnlock()
	js, ok := st.jobVMs[jobID]
	return js, ok
}

func (st *state) TrackJob(job buildkite.VmkiteJob, vmName string) {
	st.Lock()
	defer st.Unlock()
	st.jobVMs[job.ID] = jobState{
		VmkiteJob: job,
		VmName:    vmName,
	}
}

func (st *state) UntrackJob(job buildkite.VmkiteJob) {
	st.Lock()
	defer st.Unlock()
	delete(st.jobVMs, job.ID)
}

func (st *state) TrackSnapshot(job buildkite.VmkiteJob, vmName string, snapshotName string) {
	st.Lock()
	defer st.Unlock()

	st.snapshots[job.Metadata.Template] = append(st.snapshots[job.Metadata.Template], snapshot{
		vmName, snapshotName,
	})

	debugf("Tracking snapshot %s:%s for %s", vmName, snapshotName, job.Metadata.Template)
}

func (st *state) FindBestSnapshot(job buildkite.VmkiteJob) (snapshot, error) {
	st.Lock()
	defer st.Unlock()

	debugf("Finding snapshot for %v", job)
	snaps, ok := st.snapshots[job.Metadata.Template]
	if !ok || len(snaps) == 0 {
		return snapshot{}, errors.New("No snapshot found for template")
	}

	return snaps[0], nil
}
