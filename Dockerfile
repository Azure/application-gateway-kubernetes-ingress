ARG BUILDPLATFORM=linux/amd64
ARG BUILD_BASE_IMAGE
ARG BINARY_BASE_IMAGE

FROM --platform=$BUILDPLATFORM $BUILD_BASE_IMAGE AS build
WORKDIR /azure

COPY go.mod go.sum /azure/
RUN go mod download

RUN apt-get update
RUN apt-get install -y ca-certificates

ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TAG
ARG BUILD_DATE
ARG GIT_HASH

COPY cmd cmd
COPY pkg pkg
COPY Makefile Makefile

RUN make build \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    BUILD_TAG=${BUILD_TAG} \
    BUILD_DATE=${BUILD_DATE} \
    GIT_HASH=${GIT_HASH}
RUN chmod +x ./bin/appgw-ingress

#RUN ldd ./bin/appgw-ingress 2>&1 | grep 'not a dynamic executable'

FROM mcr.microsoft.com/azurelinux/base/core:3.0 AS openssl
RUN tdnf install -y openssl && tdnf clean all

FROM $BINARY_BASE_IMAGE AS final
COPY --from=openssl /usr/bin/openssl /usr/bin/openssl
COPY --from=openssl /usr/lib64/libssl.so* /usr/lib64/
COPY --from=openssl /usr/lib64/libcrypto.so* /usr/lib64/
COPY --from=build /azure/bin/appgw-ingress /appgw-ingress
CMD ["/appgw-ingress"]
