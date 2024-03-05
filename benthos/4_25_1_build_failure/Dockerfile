# BUILD step
FROM golang:1.22-alpine as builder

RUN apk add --no-progress --no-cache \
  gcc \
  musl-dev \
  linux-headers \
  git \
  openssh-client

# Switch to app dir
WORKDIR /app

COPY . ./

ARG GIT_SHA=NotAvailable
ARG GIT_BRANCH=NotAvailable

RUN go build -ldflags="-X 'main.BranchName=$GIT_BRANCH' -X 'main.SHA=$GIT_SHA'" -mod=vendor -a -tags musl -o custom-app

# PACKAGE step
FROM alpine:latest
RUN apk add --no-cache ca-certificates
RUN adduser -u 1000 -D appuser

WORKDIR /app

RUN mkdir /app/config

COPY --from=builder /app/custom-app /app/custom-app
ENTRYPOINT ["/app/custom-app"]
