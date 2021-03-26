GOOS=linux
GOARCH=amd64
VERSION := $(shell jq -r '.script_version' metadata.json)
.PHONY: build

GIT_COMMIT := $(shell git rev-list -1 HEAD)

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o reindex-linux-x64 -ldflags "-X main.gitCommit=$(GIT_COMMIT) -X main.version=$(VERSION)" main.go
	GOOS=darwin GOARCH=$(GOARCH) go build -o reindex-darwin-x64 -ldflags "-X main.gitCommit=$(GIT_COMMIT) -X main.version=$(VERSION)" main.go
