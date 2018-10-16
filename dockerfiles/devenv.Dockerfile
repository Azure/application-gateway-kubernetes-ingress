# minimal compile environment for project

FROM buildpack-deps:xenial

RUN apt-get update && apt-get -y install apt-transport-https curl

# install golang
ENV GO_VERSION 1.10.3
RUN wget -q https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz

# create gopath
RUN mkdir -p /gopath/bin

# configure env for gopath
ENV GOPATH /gopath
ENV PATH "${PATH}:${GOPATH}/bin:/usr/local/go/bin"

# get ginkgo, gomega
RUN go get github.com/onsi/ginkgo/ginkgo
RUN go get github.com/onsi/gomega/...

# get golint, goimports
RUN go get -u golang.org/x/lint/golint
RUN go get -u golang.org/x/tools/cmd/goimports

RUN apt-get clean && apt-get update && apt-get install -y locales
RUN locale-gen en_US.UTF-8

# install glide
RUN curl https://glide.sh/get | sh

WORKDIR /gopath/src/github.com/Azure/application-gateway-kubernetes-ingress