package runner

import (
	"sync"

	"github.com/macstadium/vmkite/buildkite"
)

type jobState struct {
	buildkite.VmkiteJob
	VmPath string
	VmName string
}

func (j jobState) FullPath() string {
	return j.VmPath + "/" + j.VmName
}

type state struct {
	jobVMs map[string]jobState
	sync.RWMutex
}

func (st *state) List() []jobState {
	st.RLock()
	defer st.RUnlock()
	states := []jobState{}
	for _, state := range st.jobVMs {
		states = append(states, state)
	}
	return states
}

func (st *state) Get(job buildkite.VmkiteJob) (jobState, bool) {
	st.RLock()
	defer st.RUnlock()
	js, ok := st.jobVMs[job.ID]
	return js, ok
}

func (st *state) Track(job buildkite.VmkiteJob, vmName, vmPath string) {
	st.Lock()
	defer st.Unlock()
	st.jobVMs[job.ID] = jobState{
		VmkiteJob: job,
		VmName:    vmName,
		VmPath:    vmPath,
	}
}

func (st *state) Untrack(job buildkite.VmkiteJob) {
	st.Lock()
	defer st.Unlock()
	delete(st.jobVMs, job.ID)
}
