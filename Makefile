# Docker image build and push setting
DOCKER:=docker
DOCKERFILE_DIR?=./docker

DAPR_RELEASE=1.6.0
DAPR_RUNTIME_IMAGE_NAME=daprd
DAPR_RUNTIME_DOCKER_IMAGE_TAG=pkedy/my_own_dapr:$(DAPR_RELEASE)

TARGET_OS=linux
TARGET_ARCH=amd64

OUT_DIR=dist

# build docker image for linux
BIN_PATH=$(OUT_DIR)/$(TARGET_OS)_$(TARGET_ARCH)

ifeq ($(TARGET_OS), windows)
  DOCKERFILE:=Dockerfile-windows
  BIN_PATH := $(BIN_PATH)/release
else ifeq ($(origin DEBUG), undefined)
  DOCKERFILE:=Dockerfile
  BIN_PATH := $(BIN_PATH)/release
else ifeq ($(DEBUG),0)
  DOCKERFILE:=Dockerfile
  BIN_PATH := $(BIN_PATH)/release
else
  DOCKERFILE:=Dockerfile-debug
  BIN_PATH := $(BIN_PATH)/debug
endif

ifeq ($(TARGET_ARCH),arm)
  DOCKER_IMAGE_PLATFORM:=$(TARGET_OS)/arm/v7
else ifeq ($(TARGET_ARCH),arm64)
  DOCKER_IMAGE_PLATFORM:=$(TARGET_OS)/arm64/v8
else
  DOCKER_IMAGE_PLATFORM:=$(TARGET_OS)/amd64
endif

.PHONY: release-dry-run
release-dry-run:
	goreleaser --rm-dist --skip-validate --skip-publish --snapshot

.PHONY: release
release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	goreleaser release --rm-dist

# To use buildx: https://github.com/docker/buildx#docker-ce
export DOCKER_CLI_EXPERIMENTAL=enabled

# check the required environment variables
.PHONY: check-docker-env
check-docker-env:
# ifeq ($(DAPR_REGISTRY),)
# 	$(error DAPR_REGISTRY environment variable must be set)
# endif
# ifeq ($(DAPR_TAG),)
# 	$(error DAPR_TAG environment variable must be set)
# endif

.PHONY: check-arch
check-arch:
ifeq ($(TARGET_OS),)
	$(error TARGET_OS environment variable must be set)
endif
ifeq ($(TARGET_ARCH),)
	$(error TARGET_ARCH environment variable must be set)
endif

.PHONY: build
build: check-arch
	mkdir -p $(BIN_PATH)
	CGO_ENABLED=0 GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) go build -o $(BIN_PATH)/daprd cmd/daprd/main.go

.PHONY: docker
docker: build check-docker-env check-arch
	$(info Building $(DOCKER_IMAGE_TAG) docker image ...)
ifeq ($(TARGET_ARCH),amd64)
	$(DOCKER) build --build-arg PKG_FILES=daprd -f $(DOCKERFILE_DIR)/$(DOCKERFILE) $(BIN_PATH) -t $(DAPR_RUNTIME_DOCKER_IMAGE_TAG)-$(TARGET_OS)-$(TARGET_ARCH)
else
	-$(DOCKER) buildx create --use --name daprbuild
	-$(DOCKER) run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(DOCKER) buildx build --build-arg PKG_FILES=daprd --platform $(DOCKER_IMAGE_PLATFORM) -f $(DOCKERFILE_DIR)/$(DOCKERFILE) $(BIN_PATH) -t $(DAPR_RUNTIME_DOCKER_IMAGE_TAG)-$(TARGET_OS)-$(TARGET_ARCH)
endif