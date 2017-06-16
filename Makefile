VERSION=$(shell git describe --tags --candidates=1 --dirty 2>/dev/null || echo "dev")
FLAGS=-s -w -X main.Version=$(VERSION)

vmkite: *.go buildkite/*.go cmd/*.go creator/*.go runner/*.go vsphere/*.go
	go install -a -ldflags="$(FLAGS)"
	go build -v -ldflags="$(FLAGS)"

.PHONY: clean
clean:
	rm -f vmkite

.PHONY: release
release:
	go get github.com/mitchellh/gox
	gox -ldflags="$(FLAGS)" -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -osarch="linux/amd64 darwin/amd64 windows/amd64"