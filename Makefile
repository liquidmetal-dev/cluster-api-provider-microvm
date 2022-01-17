# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

TAG ?= dev
ARCH ?= amd64
REGISTRY ?= ghcr.io
ORG ?= weaveworks
CONTROLLER_IMAGE_NAME := cluster-api-provider-microvm
CONTROLLER_IMAGE ?= $(REGISTRY)/$(ORG)/$(CONTROLLER_IMAGE_NAME)

# Directories
REPO_ROOT := $(shell git rev-parse --show-toplevel)
BIN_DIR := bin
OUT_DIR := out
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
TOOLS_SHARE_DIR := $(TOOLS_DIR)/share
MANIFEST_ROOT ?= config
CRD_ROOT ?= $(MANIFEST_ROOT)/crd/bases
WEBHOOK_ROOT ?= $(MANIFEST_ROOT)/webhook
RBAC_ROOT ?= $(MANIFEST_ROOT)/rbac

# Set --output-base for conversion-gen if we are not within GOPATH
ifneq ($(abspath $(REPO_ROOT)),$(shell go env GOPATH)/src/github.com/weaveworks/cluster-api-provider-microvm)
	GEN_OUTPUT_BASE := --output-base=$(REPO_ROOT)
else
	export GOPATH := $(shell go env GOPATH)
endif

# Set build time variables including version details
LDFLAGS := $(shell source ./hack/scripts/version.sh; version::ldflags)

PATH := $(abspath $(TOOLS_BIN_DIR)):$(PATH)

$(TOOLS_BIN_DIR):
	mkdir -p $@

$(TOOLS_SHARE_DIR):
	mkdir -p $@

$(BIN_DIR):
	mkdir -p $@

$(OUT_DIR):
	mkdir -p $@

# Binaries
COUNTERFEITER := $(TOOLS_BIN_DIR)/counterfeiter
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/controller-gen
DEFAULTER_GEN := $(TOOLS_BIN_DIR)/defaulter-gen

.DEFAULT_GOAL := help

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


##@ Linting

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint
	$(GOLANGCI_LINT) run -v --fast=false

.PHONY: test
test: ## Run tests.
	go test -v ./...

##@ Binaries

.PHONY: managers
managers: ## Build manager binary.
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS} -extldflags '-static'" -o $(BIN_DIR)/manager .


##@ Docker

.PHONY: docker-build
docker-build: docker-pull-prerequisites ## Build docker image with the manager.
	docker build --build-arg ARCH=$(ARCH) --build-arg LDFLAGS="$(LDFLAGS)" . -t $(CONTROLLER_IMAGE):$(TAG)

docker-push: ## Push docker image with the manager.
	docker push $(CONTROLLER_IMAGE):$(TAG)

.PHONY: docker-pull-prerequisites
docker-pull-prerequisites:
	docker pull docker.io/docker/dockerfile:1.1-experimental
	docker pull gcr.io/distroless/static:latest

##@ Generate

CRD_OPTIONS ?= "crd:Versions=v1"

.PHONY: generate
generate: ## Runs code generation tooling
	$(MAKE) generate-go
	$(MAKE) generate-manifests

generate-go: $(CONTROLLER_GEN) $(DEFAULTER_GEN) $(COUNTERFEITER)
	$(CONTROLLER_GEN) \
		paths=./api/... \
		object:headerFile="hack/boilerplate.go.txt" 

	$(DEFAULTER_GEN) \
		--input-dirs=./api/v1alpha1 \
		--v=0 $(GEN_OUTPUT_BASE) \
		--go-header-file=./hack/boilerplate.go.txt

	go generate ./...


generate-manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) \
		paths=./api/... \
		crd:crdVersions=v1 \
		rbac:roleName=manager-role \
		output:crd:dir=$(CRD_ROOT) \
		output:webhook:dir=$(WEBHOOK_ROOT) \
		webhook
	$(CONTROLLER_GEN) \
		paths=./controllers/... \
		output:rbac:dir=$(RBAC_ROOT) \
		rbac:roleName=manager-role
	

##@ Tools binaries

$(CONTROLLER_GEN): $(TOOLS_DIR)/go.mod # Get and build controller-gen
	cd $(TOOLS_DIR); go build -tags=tools -o $(subst hack/tools/,,$@) sigs.k8s.io/controller-tools/cmd/controller-gen

$(DEFAULTER_GEN): $(TOOLS_DIR)/go.mod # Get and build defaulter-gen
	cd $(TOOLS_DIR); go build -tags=tools -o $(subst hack/tools/,,$@) k8s.io/code-generator/cmd/defaulter-gen

$(GOLANGCI_LINT): $(TOOLS_DIR)/go.mod # Get and build golangci-lint
	cd $(TOOLS_DIR); go build -tags=tools -o $(subst hack/tools/,,$@) github.com/golangci/golangci-lint/cmd/golangci-lint

$(COUNTERFEITER): $(TOOLS_DIR)/go.mod # Get and build counterfieter
	cd $(TOOLS_DIR); go build -tags=tools -o $(subst hack/tools/,,$@) github.com/maxbrunsfeld/counterfeiter/v6

##@ Utility

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

