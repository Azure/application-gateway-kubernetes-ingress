ARG BUILDPLATFORM=linux/amd64
ARG BUILD_BASE_IMAGE

FROM --platform=$BUILDPLATFORM $BUILD_BASE_IMAGE AS build
WORKDIR /azure

COPY go.mod go.sum /azure/
RUN go mod download

RUN apt-get update
RUN apt-get install -y ca-certificates openssl

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

#RUN ldd ./bin/appgw-ingress 2>&1 | grep 'not a dynamic executable'

FROM ubuntu:20.04 AS final
COPY --from=build /azure/bin/appgw-ingress /appgw-ingress
RUN apt-get update
RUN apt-get install -y ca-certificates openssl
RUN useradd appgw-ingress-user
RUN chown appgw-ingress-user /appgw-ingress
USER appgw-ingress-user
RUN chmod +x /appgw-ingress
CMD ["/appgw-ingress"]
