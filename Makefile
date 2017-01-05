vmkite: *.go buildkite/*.go cmd/*.go vsphere/*.go
	go build

.PHONY: clean
clean:
	rm -f vmkite
