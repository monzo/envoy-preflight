FROM golang:1.20-alpine AS builder

WORKDIR /go/src/envoy-preflight

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o /go/bin/envoy-preflight ./main.go

FROM gcr.io/distroless/static-debian11

COPY --from=builder /go/bin/envoy-preflight /go/bin/envoy-preflight

ENTRYPOINT ["/go/bin/envoy-preflight"]
