# syntax=docker/dockerfile:1.2.1-labs@sha256:600e5c62eedff338b3f7a0850beb7c05866e0ef27b2d2e8c02aa468e78496ff5
FROM --platform=linux/amd64 golang:1.27-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS build

ARG PKG_NAME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN apk add --no-cache make git

WORKDIR /src
COPY go.mod ./
COPY go.sum ./

RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
	--mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
		go mod download -x

COPY . ./

RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
	--mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
		make build
RUN mv bin/${PKG_NAME}* /bin/

FROM scratch AS release-linux

ARG PKG_NAME
ARG VCS_REF
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

LABEL org.opencontainers.image.revision=$VCS_REF \
	org.opencontainers.image.source="https://github.com/hairyhenderson/${PKG_NAME}"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bin/${PKG_NAME}_${TARGETOS}-${TARGETARCH}${TARGETVARIANT} /${PKG_NAME}

ENTRYPOINT [ "/hitron_coda_exporter" ]

FROM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS alpine

ARG PKG_NAME
ARG VCS_REF
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

LABEL org.opencontainers.image.revision=$VCS_REF \
	org.opencontainers.image.source="https://github.com/hairyhenderson/${PKG_NAME}"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bin/${PKG_NAME}_${TARGETOS}-${TARGETARCH}${TARGETVARIANT} /${PKG_NAME}

ENTRYPOINT [ "/hitron_coda_exporter" ]

FROM --platform=windows/amd64 mcr.microsoft.com/windows/nanoserver:2009-KB4579311@sha256:b9d2bb3399baac0ce0881ac82f03e759dadcf4edf751ebd9682b1435f6b9b71d AS release-windows

ARG PKG_NAME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
COPY --from=build /bin/${PKG_NAME}_${TARGETOS}-${TARGETARCH}${TARGETVARIANT}.exe /${PKG_NAME}.exe

FROM release-$TARGETOS AS release
