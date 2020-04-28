FROM golang:1.13.4-alpine AS builder

WORKDIR /go/src/envoy-preflight

RUN apk update && apk add curl

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

COPY . .

RUN dep ensure

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -i -o /go/bin/envoy-preflight ./main.go

FROM gcr.io/distroless/base-debian10

COPY --from=builder /go/bin/envoy-preflight /go/bin/envoy-preflight

ENTRYPOINT /go/bin/envoy-preflight
