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

FROM --platform=$BUILDPLATFORM alpine:3.23.4
ARG ARCH TARGETOS TARGETARCH

RUN apk add --no-cache exiftool ffmpeg vips-heif vips-dev vips-tools
COPY --from=builder /go/src/github.com/robbymilo/rgallery/bin/rgallery_${TARGETOS}-${TARGETARCH} /usr/bin/rgallery

ENTRYPOINT [ "/usr/bin/rgallery" ]
CMD ["rgallery"]