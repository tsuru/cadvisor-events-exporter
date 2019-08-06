FROM golang:alpine as build
WORKDIR /go/src/app
ENV GO111MODULE=on GOPROXY=https://proxy.golang.org
RUN apk update
RUN apk add build-base linux-headers
COPY . .
RUN go install -mod=vendor ./...

FROM alpine
COPY --from=build /go/bin/* /usr/bin/

CMD [ "/usr/bin/cadvisor-local-exporter", "-logtostderr" ]
