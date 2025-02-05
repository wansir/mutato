# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:allowDangerousTypes=true"
MANIFESTS="mutations/v1alpha1"

OUTPUT_DIR=bin
ifeq (${GOFLAGS},)
	# go build with vendor by default.
	export GOFLAGS=-mod=vendor
endif

# Run go vet against code
vet: ;$(info $(M)...Begin to run go vet against code.)
	go vet ./...

# Build mutato-webhook-server binary
mutato-webhook-server: ; $(info $(M)...Begin to build mutato-webhook-server binary.)
	hack/gobuild.sh cmd/mutato-webhook-server

# Generate manifests e.g. CRD, RBAC etc.
manifests: ;$(info $(M)...Begin to generate manifests e.g. CRD, RBAC etc..)  @ ## Generate manifests e.g. CRD, RBAC etc.
	hack/generate_manifests.sh ${CRD_OPTIONS} ${MANIFESTS}
