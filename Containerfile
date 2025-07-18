FROM golang:1.24.5-alpine as builder

COPY . /usr/src/vduse-device-plugin

RUN apk add --no-cache --virtual build-dependencies build-base linux-headers

WORKDIR /usr/src/vduse-device-plugin
RUN go clean && go build cmd/vduse-dp.go

FROM alpine:3
COPY --from=builder /usr/src/vduse-device-plugin/vduse-dp /usr/bin/
WORKDIR /

RUN apk add --no-cache --virtual iproute2
LABEL io.k8s.display-name="VDUSE Device Plugin"

COPY ./entrypoint.sh /

RUN rm -rf /var/cache/apk/*

ENTRYPOINT ["/entrypoint.sh"]
