FROM golang:alpine3.11 AS builder

RUN echo http://mirrors.aliyun.com/alpine/v3.10/main/ > /etc/apk/repositories && \
    echo http://mirrors.aliyun.com/alpine/v3.10/community/ >> /etc/apk/repositories && \
    apk update && \
    apk upgrade && \
    apk add git gcc libc-dev && \
    go get github.com/Kong/go-pluginserver && \
    mkdir -p /go/src/auth-plugin

COPY . /go/src/auth-plugin
WORKDIR /go/src/auth-plugin

RUN go get -u github.com/dgrijalva/jwt-go && \
    go get -u github.com/gomodule/redigo/redis && \
    go get -u gopkg.in/yaml.v2

RUN go build -buildmode=plugin -o /tmp/auth-plugin.so plugin.go client.go codec.go config.go storage.go helper.go

#======================================= kong ==============================================

FROM kong:alpine

ENV KONG_DATABASE off
ENV KONG_GO_PLUGINS_DIR /tmp/go-plugins
ENV KONG_DECLARATIVE_CONFIG /tmp/kong.yaml
ENV KONG_PLUGINS bundled,auth-plugin
ENV KONG_PROXY_LISTEN 0.0.0.0:8000
ENV KONG_LOG_LEVEL debug

USER root

RUN mkdir /tmp/go-plugins && \
    mkdir -p /go/src/auth-plugin/

COPY kong.yml /tmp/kong.yaml
COPY conf.yaml /go/src/auth-plugin/conf.yaml
COPY --from=builder /tmp/auth-plugin.so /tmp/go-plugins/auth-plugin.so
COPY --from=builder /go/bin/go-pluginserver /usr/local/bin/go-pluginserver

RUN chmod 777 -R /tmp/ && \
    chmod 777 -R /usr/local/bin/ && \
    chmod 777 -R /go/src/auth-plugin/

USER kong