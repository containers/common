GO ?= go
GO_BUILD=$(GO) build
BUILDTAGS := containers_image_openpgp,systemd,exclude_graphdriver_devicemapper
DESTDIR ?=
PREFIX := /usr/local
CONFIGDIR := ${PREFIX}/share/containers

define go-build
	CGO_ENABLED=0 \
	GOOS=$(1) GOARCH=$(2) $(GO) build -tags "$(3)" ./...
endef

ifeq ($(shell uname -s),Linux)
define go-build-c
	CGO_ENABLED=1 \
	GOOS=$(1) GOARCH=$(2) $(GO) build -tags "$(3)" ./...
endef
else
define go-build-c
endef
endif

.PHONY:
build-cross:
	$(call go-build-c,linux) # attempt to build without tags
	$(call go-build-c,linux,,${BUILDTAGS})
	$(call go-build,linux,386,${BUILDTAGS})
	$(call go-build-c,linux) # attempt to build without tags
	$(call go-build,linux,386,${BUILDTAGS})
	$(call go-build,linux,arm,${BUILDTAGS})
	$(call go-build,linux,arm64,${BUILDTAGS})
	$(call go-build,linux,ppc64le,${BUILDTAGS})
	$(call go-build,linux,s390x,${BUILDTAGS})
	$(call go-build,darwin,amd64,${BUILDTAGS})
	$(call go-build,windows,amd64,${BUILDTAGS})
	$(call go-build,windows,386,${BUILDTAGS})
	$(call go-build,freebsd,amd64,${BUILDTAGS})
	$(call go-build,freebsd,386,${BUILDTAGS})

.PHONY: all
all: build-amd64 build-386 build-amd64-cni

.PHONY: build
build: build-amd64 build-386 build-amd64-cni

.PHONY: build-amd64
build-amd64:
	GOARCH=amd64 $(GO_BUILD) -tags $(BUILDTAGS) ./...

.PHONY: build-amd64-cni
build-amd64-cni:
	GOARCH=amd64 $(GO_BUILD) -tags $(BUILDTAGS),cni ./...

.PHONY: build-386
build-386:
ifneq ($(shell uname -s), Darwin)
	GOARCH=386 $(GO_BUILD) -tags $(BUILDTAGS) ./...
endif

.PHONY: bin/netavark-testplugin
bin/netavark-testplugin:
	$(GO_BUILD) -o $@ ./libnetwork/netavark/testplugin/

.PHONY: netavark-testplugin
netavark-testplugin: bin/netavark-testplugin

.PHONY: docs
docs:
	$(MAKE) -C docs

.PHONY: validate
validate: build/golangci-lint
	./build/golangci-lint run
	./tools/validate_seccomp.sh ./pkg/seccomp

vendor-in-container:
	podman run --privileged --rm --env HOME=/root -v `pwd`:/src -w /src golang make vendor

.PHONY: vendor
vendor:
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) mod verify

.PHONY: install.tools
install.tools: build/golangci-lint .install.md2man

build/golangci-lint: VERSION=v1.56.2
build/golangci-lint:
	curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/$(VERSION)/install.sh | sh -s -- -b ./build $(VERSION)

.install.md2man:
	$(GO) install github.com/cpuguy83/go-md2man/v2@latest

.PHONY: install
install:
	install -d ${DESTDIR}/${CONFIGDIR}
	install -m 0644 pkg/config/containers.conf ${DESTDIR}/${CONFIGDIR}/containers.conf

	$(MAKE) -C docs install

.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit: netavark-testplugin
	go test --tags seccomp,$(BUILDTAGS) -v ./...
	go test --tags remote,$(BUILDTAGS) -v ./pkg/config
	go test --tags cni,$(BUILDTAGS) -v ./libnetwork/cni

.PHONY: codespell
codespell:
	codespell --dictionary=- -w

clean: ## Clean artifacts
	$(MAKE) -C docs clean
	find . -name \*~ -delete
	find . -name \#\* -delete

.PHONY: seccomp.json
seccomp.json: $(sources)
	$(GO) run ./cmd/seccomp/generate.go
