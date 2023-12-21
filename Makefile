VERSION := $(shell git describe --tags `git rev-list --tags --max-count=1`)
PROJECT_NAME := buildenv

.phony: all build-deps build clean

all: clean build-deps build

build-deps:
	go install github.com/mitchellh/gox@latest

build:
	CGO_ENABLED=0 gox -ldflags "-X main.version=$(VERSION)" -osarch="darwin/amd64 darwin/arm64 linux/386 linux/amd64 linux/arm linux/arm64 windows/386 windows/amd64" -output "pkg/{{.OS}}_{{.Arch}}/$(PROJECT_NAME)"
	for pkg in $$(ls pkg/); do cp CONTRIBUTING.md CONTRIBUTORS.md LICENSE NOTICE pkg/$${pkg}; done
	for pkg in $$(ls pkg/); do cd pkg/$${pkg}; tar cvzf "../../$(PROJECT_NAME)-$${pkg}-$(VERSION).tar.gz" $(PROJECT_NAME)*; cd ../..; done

clean:
	rm -f *.tar.gz
	rm -rf pkg
