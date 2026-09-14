FROM --platform=$BUILDPLATFORM node:24.12.0-alpine3.22 AS frontend
ARG ARCH TARGETOS TARGETARCH

RUN apk add --no-cache make git
WORKDIR /usr/app
COPY . .
RUN npm i
RUN make assets

FROM --platform=$BUILDPLATFORM golang:1.25.10-alpine3.23 AS builder
ARG ARCH TARGETOS TARGETARCH

RUN apk update && apk add --no-cache make build-base git gcc g++

RUN ls -al /usr/local/go

WORKDIR /go/src/github.com/robbymilo/rgallery
COPY . /go/src/github.com/robbymilo/rgallery

COPY --from=frontend /usr/app/pkg/dist ./pkg/dist

RUN make build-rgallery TARGETOS=${TARGETOS} TARGETARCH=${TARGETARCH}

FROM debian:trixie-slim AS runtime-deps
ARG TARGETARCH

# Use Debian FFmpeg with AMD and Intel VA-API drivers, including older Intel GPUs.
RUN sed -i 's/^Components: main$/Components: main non-free/' /etc/apt/sources.list.d/debian.sources \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libimage-exiftool-perl ffmpeg libvips-tools mesa-va-drivers vainfo \
    && if [ "$TARGETARCH" = "amd64" ]; then apt-get install -y --no-install-recommends intel-media-va-driver-non-free i965-va-driver; fi \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /media /cache /data /config \
    && chown 1000:1000 /media /cache /data /config

FROM runtime-deps AS runtime
ARG TARGETOS TARGETARCH
ENV MESA_SHADER_CACHE_DIR=/cache/mesa MESA_SHADER_CACHE_MAX_SIZE=64M
COPY --from=builder /go/src/github.com/robbymilo/rgallery/bin/rgallery_${TARGETOS}-${TARGETARCH} /usr/bin/rgallery

ENTRYPOINT [ "/usr/bin/rgallery" ]
CMD []
