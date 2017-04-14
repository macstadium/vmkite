vmkite: *.go buildkite/*.go cmd/*.go creator/*.go runner/*.go vsphere/*.go
	go install -a
	go build -v

.PHONY: clean
clean:
	rm -f vmkite

.PHONY: release
release:
	go get github.com/mitchellh/gox
	gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -osarch="linux/amd64 darwin/amd64 windows/amd64"