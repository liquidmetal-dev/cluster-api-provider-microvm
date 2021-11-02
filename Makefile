# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

TAG ?= dev
ARCH ?= amd64
CONTROLLER_IMAGE_NAME := cluster-api-microvm-controller
# Todo add registry prefix
CONTROLLER_IMAGE ?= $(CONTROLLER_IMAGE_NAME)

# Directories
REPO_ROOT := $(shell git rev-parse --show-toplevel)
BIN_DIR := bin
OUT_DIR := out
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
TOOLS_SHARE_DIR := $(TOOLS_DIR)/share

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
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint
	$(GOLANGCI_LINT) run -v --fast=false

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

##@ Docker

.PHONY: docker-build
docker-build: docker-pull-prerequisites ## Build docker image with the manager.
	docker build --build-arg ARCH=$(ARCH) --build-arg LDFLAGS="$(LDFLAGS)" . -t $(CONTROLLER_IMAGE)-$(ARCH):$(TAG)

docker-push: ## Push docker image with the manager.
	docker push $(CONTROLLER_IMAGE)-$(ARCH):$(TAG)

.PHONY: docker-pull-prerequisites
docker-pull-prerequisites:
	docker pull docker.io/docker/dockerfile:1.1-experimental
	docker pull gcr.io/distroless/static:latest

##@ Generate

CRD_OPTIONS ?= "crd:Versions=v1"

.PHONY: generate
generate: ## Runs code generation tooling
	$(MAKE) generate-api

generate-api: controller-gen defaulter-gen
	$(CONTROLLER_GEN) \
		paths=./api/... \
		output:crd:artifacts:config=config/crd/bases \
		object:headerFile="hack/boilerplate.go.txt" \
		crd:crdVersions=v1 \
		rbac:roleName=manager-role \
		webhook

	$(DEFAULTER_GEN) \
		--input-dirs=./api/v1alpha1 \
		--v=0 $(GEN_OUTPUT_BASE) \
		--go-header-file=./hack/boilerplate.go.txt


##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

DEFAULTER_GEN = $(shell pwd)/bin/defaulter-gen
defaulter-gen: ## Download defaulter-gen locally if necessary.
	$(call go-get-tool,$(DEFAULTER_GEN),k8s.io/code-generator/cmd/defaulter-gen@v0.22.2)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

##@ Tools binaries

$(GOLANGCI_LINT): $(TOOLS_DIR)/go.mod # Get and build golangci-lint
	cd $(TOOLS_DIR); go build -tags=tools -o $(subst hack/tools/,,$@) github.com/golangci/golangci-lint/cmd/golangci-lint