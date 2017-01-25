vmkite: *.go buildkite/*.go cmd/*.go creator/*.go runner/*.go vsphere/*.go
	go build

.PHONY: clean
clean:
	rm -f vmkite
