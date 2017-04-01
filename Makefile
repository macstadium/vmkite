vmkite: *.go buildkite/*.go cmd/*.go creator/*.go runner/*.go vsphere/*.go
	go build

.PHONY: clean
clean:
	rm -f vmkite

.PHONY: release
release:
	go get github.com/mitchellh/gox
	gox -osarch="linux/amd64 darwin/amd64 windows/amd64 windows/386"