vmkite: *.go buildkite/*.go cmd/*.go creator/*.go runner/*.go vsphere/*.go
	go build

.PHONY: clean
clean:
	rm -f vmkite

.PHONY: release
release:
	go get github.com/mitchellh/gox
	gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -os="linux windows darwin"