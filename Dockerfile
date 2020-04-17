FROM golang:buster AS build
COPY . /app
WORKDIR /app
RUN go get -d
RUN go test -test.timeout 30s 
RUN CGO_ENABLED=0 go build -o scuttle

FROM scratch
COPY --from=build /app/scuttle /scuttle
