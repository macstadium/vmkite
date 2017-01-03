vmkite
======

Objective
---------

Given a VMware vSphere cluster of Mac hardware, monitor a [Buildkite][bk]
account for jobs and rapidly launch a suitable clean-state macOS virtual
machine running [Buildkite Agent][bka] for each one.

Strategy
--------

* Assume base images / VMs / templates loaded into VMware cluster.
* Poll Buildkite API for jobs matching `vmkite-name=X` where X is a known base/template.
* Check for available slots based on X virtual machines per host.
* Create VM with independent non-persistent disk from base image.
* VM launches Buildkite Agent with `vmkite-name=X` metadata.
* After a job, the VM shuts itself down.

Tools
-----

[`makemac.go`][makemac] by Brad Fitzpatrick for `golang/build` is a rough-draft
quality tool with similar objectives for building the Go codebase on macOS VMs.
It wraps the `govc` CLI interface of the `govmomi` VMware Go library. It has a
few [code review comments][makemac-gerrit].

[`govmomi`][govmomi] is the official Go library for the VMware vSphere API. It
has a CLI interface called [`govc`][govc].

[`go-buildkite`][go-buildkite] is the Buildkite API client for Go.

Notes
-----

`makemac.go` code review notes suggest booting from a template is slow, and
that creating new VMs from scratch and attaching a frozen disk image is
fastest.


[bk]: https://buildkite.com/
[bka]: https://github.com/buildkite/agent
[go-buildkite]: https://github.com/buildkite/go-buildkite
[govc]: https://github.com/vmware/govmomi/tree/master/govc
[govmomi]: https://github.com/vmware/govmomi
[makemac-gerrit]: https://go-review.googlesource.com/#/c/28584/
[makemac]: https://github.com/golang/build/blob/master/cmd/makemac/makemac.go
