export GO111MODULE=off

GO ?= go
GO_BUILD=$(GO) build
# Go module support: set `-mod=vendor` to use the vendored sources
ifeq ($(shell go help mod >/dev/null 2>&1 && echo true), true)
	GO_BUILD=GO111MODULE=on $(GO) build -mod=vendor
endif
DESTDIR ?=
PROJECT := github.com/containers/common

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

.PHONY: docs
docs:
	$(MAKE) -C docs

.PHONY: validate
validate:
	golangci-lint run

.PHONY: vendor
vendor:
	export GO111MODULE=on \
		$(GO) mod tidy && \
		$(GO) mod vendor && \
		$(GO) mod verify

.PHONY: install.tools
install.tools: .install.golangci-lint .install.md2man

.install.golangci-lint:
	if [ ! -x "$(GOBIN)/golangci-lint" ]; then \
		curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOBIN)/ v1.18.0; \
	fi


.install.md2man:
	if [ ! -x "$(GOBIN)/go-md2man" ]; then \
		   $(call go-get,github.com/cpuguy83/go-md2man); \
	fi

.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit:
	go test -v $(PROJECT)/pkg/...

clean: ## Clean artifacts
	$(MAKE) -C docs clean
	rm -rf \
		bin
	find . -name \*~ -delete
	find . -name \#\* -delete