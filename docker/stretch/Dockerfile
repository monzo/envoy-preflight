FROM golang:stretch AS build
COPY . /app
WORKDIR /app
RUN go get -d
RUN CGO_ENABLED=0 go build -o scuttle

FROM debian:stretch AS final
WORKDIR /app
COPY --from=build /app/scuttle /bin/scuttle
ENTRYPOINT [ "scuttle" ]