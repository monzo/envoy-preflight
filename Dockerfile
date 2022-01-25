FROM golang:1.17-buster AS build
ARG VERSION="local"
COPY . /app
WORKDIR /app
RUN go get -d
RUN go test -test.timeout 30s 
RUN CGO_ENABLED=0 go build -o scuttle -ldflags="-X 'main.Version=${VERSION}'"

FROM scratch
COPY --from=build /app/scuttle /scuttle
