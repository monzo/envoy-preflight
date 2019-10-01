FROM golang AS build
COPY . /app
WORKDIR /app
RUN go get -d
RUN go build -o scuttle

FROM alpine:latest AS final
# libc from the build stage is not the same as the alpine libc
# create a symlink to where it expects it since they are compatable. https://stackoverflow.com/a/35613430/3105368
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
WORKDIR /app
COPY --from=build /app/scuttle /bin/scuttle
ENTRYPOINT [ "scuttle" ]