vmkite
======

Objective
---------

Given a VMware ESXi cluster, maintain a pool of macOS virtual machines each
in a clean state waiting to perform CI builds via buildkite-agent.

Strategy
--------

* Load a base image / VM / template into ESXi cluster.
* Initialize the pool by creating the desired number of VMs (minus already running VMs).
    * New VM with existing disk image? New from template? Clone running VM?
    * Optimization: Instant Clone? (vmfork / project fargo)
* Each VM starts `buildkite-agent` on boot, connecting and waiting for jobs.
* After a job, the VM shuts itself down.
    * Optimization: reverts to snapshot of clean state.
* Poll running VM count, create new VMs when fewer than desired.
    * Optimization: VMware events / hooks instead of polling?

Tools
-----

[`makemac.go`][makemac] by Brad Fitzpatrick for `golang/build` has very similar objectives, but for building the Go codebase as opposed to Buildkite agents. It wraps the `govc` CLI interface of the `govmomi` VMware Go library and is of rough-draft quality. It has a few [code review comments][makemac-gerrit].

[`govmomi`][govmomi] is the official Go library for the VMware vSphere API. It has a CLI interface called [`govc`][govc].

Notes
-----

`makemac.go` code review notes suggest booting from a template is slow, and that creating new VMs from scratch and attaching a frozen disk image is fastest.


[govc]: https://github.com/vmware/govmomi/tree/master/govc
[govmomi]: https://github.com/vmware/govmomi
[makemac]: https://github.com/golang/build/blob/master/cmd/makemac/makemac.go
[makemac-gerrit]: https://go-review.googlesource.com/#/c/28584/
