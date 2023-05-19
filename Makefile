ORG_PATH := github.com/Azure
PROJECT_NAME := application-gateway-kubernetes-ingress
REPO_PATH = ${ORG_PATH}/${PROJECT_NAME}

VERSION_VAR = ${REPO_PATH}/pkg/version.Version
BUILD_TAG ?= $(shell git describe --abbrev=0 --tags)

DATE_VAR = ${REPO_PATH}/pkg/version.BuildDate
BUILD_DATE ?= $(shell date +%Y-%m-%d-%H:%MT%z)

COMMIT_VAR = ${REPO_PATH}/pkg/version.GitCommit
GIT_HASH ?= $(shell git rev-parse --short HEAD)

GO_BINARY_NAME ?= appgw-ingress
GOOS ?= linux
GARCH ?= arm64

BUILD_BASE_IMAGE ?= golang:1.17.3

REPO ?= appgwreg.azurecr.io
IMAGE_NAME = public/azure-application-gateway/kubernetes-ingress-staging
IMAGE = ${REPO}/${IMAGE_NAME}

IMAGE_RESULT_FLAG = --output=type=oci,dest=$(shell pwd)/image/ingress-agic-$(VERSION).tar
ifeq ($(PUSH_IMAGE), true)
	IMAGE_RESULT_FLAG = --push
endif

ifeq ($(RELEASE_IMAGE), true)
	IMAGE_NAME = public/azure-application-gateway/kubernetes-ingress
endif

TAG_LATEST ?= false

ifeq ($(TAG_LATEST), true)
	IMAGE_TAGS = \
		--tag $(IMAGE):$(BUILD_TAG) \
		--tag $(IMAGE):latest
else
	IMAGE_TAGS = \
		--tag $(IMAGE):$(BUILD_TAG)
endif

# Platforms to build the multi-arch image for.
IMAGE_PLATFORMS ?= linux/amd64,linux/arm64

GO_BUILD_VARS = \
	${REPO_PATH}/pkg/version.Version=${BUILD_TAG} \
	${REPO_PATH}/pkg/version.BuildDate=${BUILD_DATE} \
	${REPO_PATH}/pkg/version.GitCommit=${GIT_HASH}

GO_LDFLAGS := -s -w $(patsubst %,-X %, $(GO_BUILD_VARS))

build-image-multi-arch:
	@mkdir -p $(shell pwd)/image
	@docker run --rm --privileged linuxkit/binfmt:v0.8
	@docker buildx build $(IMAGE_RESULT_FLAG) \
		--platform $(IMAGE_PLATFORMS) \
		--build-arg "BUILD_BASE_IMAGE=$(BUILD_BASE_IMAGE)" \
		--build-arg "BUILD_TAG=$(BUILD_TAG)" \
		--build-arg "BUILD_DATE=$(BUILD_DATE)" \
		--build-arg "GIT_HASH=$(GIT_HASH)" \
		$(IMAGE_TAGS) \
		$(shell pwd)

build:
	go build -mod=readonly -v -ldflags="$(GO_LDFLAGS)" -v -o ./bin/${GO_BINARY_NAME} ./cmd/appgw-ingress

lint-all: lint lint-helm

lint:
	@go get -u golang.org/x/lint/golint
	@golint $(go list ./... | grep -v /vendor/) | tee /tmp/lint.out
	@if [ -s /tmp/lint.out ]; then \
		echo "\e[101;97m golint FAILED \e[0m"; \
		exit 1; \
	fi
	@echo "\e[42;97m golint SUCCEEDED \e[0m"

lint-helm:
	helm lint ./helm/ingress-azure

vet-all: vet vet-unittest vet-e2e

vet:
	@echo "Vetting controller source code"
	@if go vet -v ./...; then \
		echo "\e[42;97m govet SUCCEEDED \e[0m"; \
	else \
		echo "\e[101;97m govet FAILED \e[0m"; \
		exit 1; \
	fi

vet-unittest:
	@echo "Vetting test source code"
	@if go vet -v -tags=unittest ./...; then \
		echo "\e[42;97m govet SUCCEEDED \e[0m"; \
	else \
		echo "\e[101;97m govet FAILED \e[0m"; \
		exit 1; \
	fi

vet-e2e:
	@echo "Vetting e2e source code"
	@cd ./scripts/e2e
	@if go vet -v -tags=e2e ./...; then \
		echo "\e[42;97m govet SUCCEEDED \e[0m"; \
	else \
		echo "\e[101;97m govet FAILED \e[0m"; \
		exit 1; \
	fi
	@cd ../..

test-all: unittest

unittest:
	@go install github.com/jstemmer/go-junit-report@latest
	@go install github.com/axw/gocov/gocov@latest
	@go install github.com/AlekSi/gocov-xml@latest
	@go install github.com/matm/gocov-html/cmd/gocov-html@latest
	@go install github.com/onsi/ginkgo/v2
	@go mod tidy
	@go test -timeout 80s -v -coverprofile=coverage.txt -covermode count -tags unittest ./... > testoutput.txt || { echo "go test returned non-zero"; cat testoutput.txt; exit 1; }
	@cat testoutput.txt | go-junit-report > report.xml
	@gocov convert coverage.txt > coverage.json
	@gocov-xml < coverage.json > coverage.xml
	@mkdir coverage
	@gocov-html < coverage.json > coverage/index.html