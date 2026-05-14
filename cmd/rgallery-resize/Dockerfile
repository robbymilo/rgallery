FROM --platform=$BUILDPLATFORM golang:1.25.10-alpine3.23 AS builder
ARG ARCH TARGETOS TARGETARCH

RUN apk update && apk add --no-cache exiftool make build-base git gcc g++ libgcc

WORKDIR /go/src/github.com/robbymilo/rgallery-resize
COPY . /go/src/github.com/robbymilo/rgallery-resize

RUN make build-rgallery-resize TARGETOS=${TARGETOS} TARGETARCH=${TARGETARCH}

FROM --platform=$BUILDPLATFORM alpine:3.18.5
ARG ARCH TARGETOS TARGETARCH

RUN apk add --no-cache libc6-compat vips-heif vips-dev vips-tools
COPY --from=builder /go/src/github.com/robbymilo/rgallery-resize/bin/rgallery-resize_${TARGETOS}-${TARGETARCH} /usr/bin/rgallery-resize

ENTRYPOINT [ "/usr/bin/rgallery-resize" ]
CMD ["rgallery-resize"]
