FROM golang AS build
COPY . /app
WORKDIR /app
RUN go get -d
RUN CGO_ENABLED=0 go build -o scuttle

FROM alpine:latest AS final
WORKDIR /app
COPY --from=build /app/scuttle /bin/scuttle
ENTRYPOINT [ "scuttle" ]