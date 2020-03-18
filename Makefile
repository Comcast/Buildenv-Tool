VERSION := $(shell cat version.txt)
PROJECT_NAME := buildenv

.phony: all build-deps build clean

all: clean build-deps build

build-deps:
	go get github.com/mitchellh/gox
	go get github.com/aktau/github-release

build:
	CGO_ENABLED=0 gox -ldflags "-X main.version=$(VERSION)" -os "linux darwin windows" -arch "386 amd64" -output "pkg/{{.OS}}_{{.Arch}}/$(PROJECT_NAME)"
	for pkg in $$(ls pkg/); do cp CONTRIBUTING.md CONTRIBUTORS.md LICENSE NOTICE pkg/$${pkg}; done
	for pkg in $$(ls pkg/); do cd pkg/$${pkg}; tar cvzf "../../$(PROJECT_NAME)-$${pkg}-$(VERSION).tar.gz" $(PROJECT_NAME)*; cd ../..; done

clean:
	rm -f *.tar.gz
	rm -rf pkg
