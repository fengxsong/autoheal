APP_NAME ?= autoheal

BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
BUILD_USER = $(shell whoami)
VERSION ?= $(shell git describe --dirty --tags --always)
REVISION = $(shell git rev-parse --short HEAD)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GO ?= go
BUILDAH ?= buildah

BUILD_PATH = ./cmd/autoheal/
OUTPUT_PATH = build/_output/bin

LDFLAGS := -s -X github.com/prometheus/common/version.Version=${VERSION} \
	-X github.com/prometheus/common/version.Revision=${REVISION} \
	-X github.com/prometheus/common/version.Branch=${BRANCH} \
	-X github.com/prometheus/common/version.BuildUser=${BUILD_USER} \
	-X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}

IMAGE_REPO ?= fengxsong/${APP_NAME}
IMAGE_TAG ?= ${REVISION}
IMAGE := ${IMAGE_REPO}:${IMAGE_TAG}

dep:
	go mod tidy

fmt:
	go fmt ./...

clean:
	rm -rf bin/

manifest: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager crd paths=./pkg/apis/autoheal/v1alpha2/... output:crd:dir=./manifests/crds

generate:
	bash hack/update-codegen.sh

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.3
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

.PHONY: bin/autoheal
bin/autoheal.%:
	GOOS=$(word 2,$(subst ., ,$@)) GOARCH=$(word 3,$(subst ., ,$@)) CGO_ENABLED=0 ${GO} build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o $@ ${BUILD_PATH}
local-cross: clean bin/autoheal.linux.amd64 bin/autoheal.linux.arm64

upx: local-cross
	upx bin/autoheal.linux.amd64

# Build multiarch container images
buildimages:
	# Make sure qemu-user-static+binfmt-support is installed
	update-binfmts --enable
	${BUILDAH} manifest create ${APP_NAME}
	${BUILDAH} build --manifest ${APP_NAME} --arch amd64 --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 --build-arg LDFLAGS="${LDFLAGS}" -t ${IMAGE} -t ${IMAGE_REPO}:latest -f Dockerfile .
	${BUILDAH} build --manifest ${APP_NAME} --arch arm64 --build-arg TARGETOS=linux --build-arg TARGETARCH=arm64 --build-arg LDFLAGS="${LDFLAGS}" -t ${IMAGE} -t ${IMAGE_REPO}:latest -f Dockerfile .

# Push the images
pushimages:
	${BUILDAH} manifest push --all ${APP_NAME} docker://docker.io/${IMAGE}