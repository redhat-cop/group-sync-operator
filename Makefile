SHELL := /bin/bash
# Current Operator version
VERSION ?= 0.0.1
# Operator SDK
OPERATOR_SDK ?= operator-sdk
# YQ Version
YQ_VERSION ?= 3.2.1
# Platform
ifeq ($(OS),Windows_NT)
  PLATFORM := windows
else
  UNAME_S = $(shell uname -s)
  ifeq ($(UNAME_S),Linux)
    PLATFORM := linux
  endif
  ifeq ($(UNAME_S),Darwin)
    PLATFORM := darwin
  endif
endif
# Default bundle image tag
BUNDLE_IMG ?= quay.io/redhat-cop/group-sync-operator-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= quay.io/redhat-cop/group-sync-operator:$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,crdVersions=v1beta1,preserveUnknownFields=false"

OPERATOR_NAME ?=$(shell basename `pwd`)

CHART_REPO_URL ?= http://example.com
HELM_REPO_DEST ?= /tmp/gh-pages

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

BUILD_TOOLS_DIR=$(shell pwd)/buildtools
yq:
    ifeq ("$(wildcard $(BUILD_TOOLS_DIR)/yq)","")
	@{ \
	set -e ;\
	mkdir -p $(BUILD_TOOLS_DIR) ;\
	curl -L https://github.com/mikefarah/yq/releases/download/3.3.0/yq_$(PLATFORM)_amd64 --output $(BUILD_TOOLS_DIR)/yq && chmod +x $(BUILD_TOOLS_DIR)/yq ;\
	}
    endif
YQ=$(BUILD_TOOLS_DIR)/yq

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: kustomize
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen yq
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	YQ=$(YQ) $(shell pwd)/hack/fix-ldap-provider-crd.sh

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	$(OPERATOR_SDK)  generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Generate helm chart
helmchart: kustomize
	mkdir -p ./charts/${OPERATOR_NAME}/templates
	cp ./config/helmchart/templates/* ./charts/${OPERATOR_NAME}/templates
	$(KUSTOMIZE) build ./config/helmchart | sed 's/release-namespace/{{.Release.Namespace}}/' > ./charts/${OPERATOR_NAME}/templates/rbac.yaml
	version=${VERSION} envsubst < ./config/helmchart/Chart.yaml.tpl  > ./charts/${OPERATOR_NAME}/Chart.yaml
	version=${VERSION} image_repo=$${IMG%:*} envsubst < ./config/helmchart/values.yaml.tpl  > ./charts/${OPERATOR_NAME}/values.yaml
	helm lint ./charts/${OPERATOR_NAME}	

helmchart-repo: helmchart
	mkdir -p ${HELM_REPO_DEST}/${OPERATOR_NAME}
	helm package -d ${HELM_REPO_DEST}/${OPERATOR_NAME} ./charts/${OPERATOR_NAME}
	helm repo index --url ${CHART_REPO_URL} ${HELM_REPO_DEST}

helmchart-repo-push: helmchart-repo	
	git -C ${HELM_REPO_DEST} add .
	git -C ${HELM_REPO_DEST} status
	git -C ${HELM_REPO_DEST} commit -m "Release ${VERSION}"
	git -C ${HELM_REPO_DEST} push origin "gh-pages"	
