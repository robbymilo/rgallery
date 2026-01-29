FROM node:24.12.0-alpine

RUN apk update && apk add --no-cache curl git tar make bash gcompat libc6-compat

ARG HUGO_VERSION
ENV HUGO_VERSION=$HUGO_VERSION

RUN npm i -g netlify-cli@18.0.4

RUN curl -sSL https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_Linux-64bit.tar.gz | tar -v -C /usr/local/bin -xz && \
   chmod 755 /usr/local/bin/hugo && \
   /usr/local/bin/hugo version
