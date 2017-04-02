vmkite
======

Provides a daemon that listens for builds on [Buildkite][bk] and spawns ephemeral virtual machines running [Buildkite Agent][bka] on a VMWare vSphere cluster. After the build has finished the instance is destroyed. This allows for a repeatable environment for testing.

Requires a vSphere Datastore with the base virtual machine image in a `.vmx` format. Currently MacStadium provides these for you as part of your private cluster, contact us for the location to use. 

**Note: currently only supports macOS virtual machines**

Running
-------

Either install with `go get github.com/macstadium/vmkite` or install one of the releases for your platform.

```bash
vmkite run \
  --buildkite-agent-token=BUILDKITE-AGENT-TOKEN \
  --buildkite-api-token=BUILDKITE-API-TOKEN \
  --buildkite-org=BUILDKITE-ORG \
  --target-datastore=TARGET-DATASTORE \
  --source-datastore=SOURCE-DATASTORE \
  --vm-cluster-path=VM-CLUSTER-PATH \
  --vm-network-label=VM-NETWORK-LABEL \
  --vm-memory-mb=VM-MEMORY-MB \
  --vm-num-cpus=VM-NUM-CPUS \
  --vm-num-cores-per-socket=VM-NUM-CORES-PER-SOCKET
```

Strategy
--------

* Assume base images / VMs / templates loaded into VMware cluster.
* Poll Buildkite API for jobs matching `vmkite-name=X` where X is a known base/template.
* Check for available slots based on X virtual machines per host.
* Create VM with independent non-persistent disk from base image.
* VM launches Buildkite Agent with `vmkite-name=X` metadata.
* After a job, the VM shuts itself down.

Notes
-----

Initially derived from [`makemac.go`][makemac] by Brad Fitzpatrick for `golang/build` is a rough-draft
quality tool with similar objectives for building the Go codebase on macOS VMs.
It wraps the `govc` CLI interface of the `govmomi` VMware Go library. It has a
few [code review comments][makemac-gerrit].

`makemac.go` code review notes suggest booting from a template is slow, and
that creating new VMs from scratch and attaching a frozen disk image is
fastest.

[`govmomi`][govmomi] is the official Go library for the VMware vSphere API. It
has a CLI interface called [`govc`][govc].

[`go-buildkite`][go-buildkite] is the Buildkite API client for Go.

[bk]: https://buildkite.com/
[bka]: https://github.com/buildkite/agent
[go-buildkite]: https://github.com/buildkite/go-buildkite
[govc]: https://github.com/vmware/govmomi/tree/master/govc
[govmomi]: https://github.com/vmware/govmomi
[makemac-gerrit]: https://go-review.googlesource.com/#/c/28584/
[makemac]: https://github.com/golang/build/blob/master/cmd/makemac/makemac.go
