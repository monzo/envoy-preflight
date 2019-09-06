# Build container
FROM golang:alpine AS builder

RUN apk -v --update add --no-cache \
    git

WORKDIR /build

ENV CGO_ENABLED=0
ENV GOOS=linux

ADD . .

RUN go mod init scratch
RUN go build -o /envoy-preflight

# -----

# Runtime container
FROM scratch

COPY --from=builder /envoy-preflight /bin/envoy-preflight
