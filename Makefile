BUILD_COMMIT := $(shell ./scripts/build/get-build-commit.sh)
BUILD_TIMESTAMP := $(shell ./scripts/build/get-build-timestamp.sh)
BUILD_HOSTNAME := $(shell ./scripts/build/get-build-hostname.sh)

LDFLAGS := "-X github.com/redhat-cop/group-sync-operator/version.Version=$(VERSION) \
	-X github.com/redhat-cop/group-sync-operator/version.Vcs=$(BUILD_COMMIT) \
	-X github.com/redhat-cop/group-sync-operator/version.Timestamp=$(BUILD_TIMESTAMP) \
	-X github.com/redhat-cop/group-sync-operator/version.Hostname=$(BUILD_HOSTNAME)"

all: operator

# Build manager binary
operator: generate fmt vet
	go build -o build/_output/bin/group-sync-operator  -ldflags $(LDFLAGS) github.com/redhat-cop/group-sync-operator/cmd/manager

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Test
test: generate fmt vet
	go test ./pkg/... ./cmd/... -coverprofile cover.out