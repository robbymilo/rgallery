FROM --platform=$BUILDPLATFORM golang:1.25.10-alpine3.23 AS builder
ARG ARCH TARGETOS TARGETARCH

RUN apk update && apk add --no-cache exiftool make build-base git

WORKDIR /go/src/github.com/robbymilo/rgallery-geo
COPY . /go/src/github.com/robbymilo/rgallery-geo

RUN make build-rgallery-geo TARGETOS=${TARGETOS} TARGETARCH=${TARGETARCH}

FROM --platform=$BUILDPLATFORM alpine:3.23.4
ARG ARCH TARGETOS TARGETARCH

RUN apk add --no-cache libc6-compat
COPY --from=builder /go/src/github.com/robbymilo/rgallery-geo/bin/rgallery-geo_${TARGETOS}-${TARGETARCH} /usr/bin/rgallery-geo

ENTRYPOINT [ "/usr/bin/rgallery-geo" ]
CMD ["rgallery-geo"]
