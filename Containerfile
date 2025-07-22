FROM golang:1.24.5-alpine as builder
RUN apk add --no-cache --virtual build-dependencies build-base linux-headers

COPY . /usr/src/vduse-device-plugin
WORKDIR /usr/src/vduse-device-plugin
RUN go clean && go build cmd/vduse-dp.go

FROM alpine:3
LABEL io.k8s.display-name="VDUSE Device Plugin"
RUN apk add --no-cache --virtual iproute2
RUN rm -rf /var/cache/apk/*

COPY --from=builder /usr/src/vduse-device-plugin/vduse-dp /usr/bin/
WORKDIR /
COPY ./entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
