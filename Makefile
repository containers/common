export GO111MODULE=off

GO ?= go
GO_BUILD=$(GO) build
# Go module support: set `-mod=vendor` to use the vendored sources
ifeq ($(shell go help mod >/dev/null 2>&1 && echo true), true)
	GO_BUILD=GO111MODULE=on $(GO) build -mod=vendor
endif
BUILDTAGS := containers_image_openpgp,systemd,no_libsubid
DESTDIR ?=
PREFIX := /usr/local
CONFIGDIR := ${PREFIX}/share/containers
PROJECT := github.com/containers/common

# Enforce the GOPROXY to make sure dependencies are resovled consistently
# across different environments.
export GOPROXY := https://proxy.golang.org

# If GOPATH not specified, use one in the local directory
ifeq ($(GOPATH),)
export GOPATH := $(CURDIR)/_output
unexport GOBIN
endif
FIRST_GOPATH := $(firstword $(subst :, ,$(GOPATH)))
GOPKGDIR := $(FIRST_GOPATH)/src/$(PROJECT)
GOPKGBASEDIR ?= $(shell dirname "$(GOPKGDIR)")

GOBIN := $(shell $(GO) env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(FIRST_GOPATH)/bin
endif

define go-get
	env GO111MODULE=off \
		$(GO) get -u ${1}
endef

define go-build
	GOOS=$(1) GOARCH=$(2) $(GO) build -tags "$(3)" ./...
endef

.PHONY:
build-cross:
	$(call go-build,linux,386,${BUILDTAGS})
	$(call go-build,linux,arm,${BUILDTAGS})
	$(call go-build,linux,arm64,${BUILDTAGS})
	$(call go-build,linux,ppc64le,${BUILDTAGS})
	$(call go-build,linux,s390x,${BUILDTAGS})
	$(call go-build,darwin,amd64,${BUILDTAGS})
	$(call go-build,windows,amd64,remote ${BUILDTAGS})
	$(call go-build,windows,386,remote ${BUILDTAGS})

.PHONY: all
all: build-amd64 build-386

.PHONY: build
build: build-amd64 build-386

.PHONY: build-amd64
build-amd64:
	GOARCH=amd64 $(GO_BUILD) -tags $(BUILDTAGS) ./...

.PHONY: build-386
build-386:
ifneq ($(shell uname -s), Darwin)
	GOARCH=386 $(GO_BUILD) -tags $(BUILDTAGS) ./...
endif

.PHONY: docs
docs:
	$(MAKE) -C docs

.PHONY: validate
validate: build/golangci-lint
	./build/golangci-lint run --build-tags no_libsubid
	./tools/validate_seccomp.sh ./pkg/seccomp

vendor-in-container:
	podman run --privileged --rm --env HOME=/root -v `pwd`:/src -w /src golang make vendor

.PHONY: vendor
vendor:
	GO111MODULE=on $(GO) mod tidy
	GO111MODULE=on $(GO) mod vendor
	GO111MODULE=on $(GO) mod verify

.PHONY: install.tools
install.tools: build/golangci-lint .install.md2man

build/golangci-lint:
	export \
		VERSION=v1.30.0 \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		BINDIR=build && \
	curl -sfL $$URL/$$VERSION/install.sh | sh -s $$VERSION


.install.md2man:
	if [ ! -x "$(GOBIN)/go-md2man" ]; then \
		   $(call go-get,github.com/cpuguy83/go-md2man); \
	fi

.PHONY: install
install:
	install -d ${DESTDIR}/${CONFIGDIR}
	install -m 0644 pkg/config/containers.conf ${DESTDIR}/${CONFIGDIR}/containers.conf

	$(MAKE) -C docs install

.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit:
	go test --tags $(BUILDTAGS) -v ./libimage
	go test --tags $(BUILDTAGS) -v $(PROJECT)/pkg/...
	go test --tags remote,seccomp,$(BUILDTAGS) -v $(PROJECT)/pkg/...

.PHONY: codespell
codespell:
	codespell -S bin,vendor,.git,go.sum,changelog.txt,.cirrus.yml,"RELEASE_NOTES.md,*.xz,*.gz,*.tar,*.tgz,bin2img,*ico,*.png,*.1,*.5,copyimg,*.orig,apidoc.go" -L uint,iff,od,seeked,splitted,marge,ERROR,hist,ether -w

clean: ## Clean artifacts
	$(MAKE) -C docs clean
	find . -name \*~ -delete
	find . -name \#\* -delete

.PHONY: seccomp.json
seccomp.json: $(sources)
	$(GO) run ./cmd/seccomp/generate.go
