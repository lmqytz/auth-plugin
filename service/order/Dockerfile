FROM golang:alpine3.11

LABEL maintainer="luoman <luoman.tz@gmail.com>"

WORKDIR /go

ENV GO111MODULE on
ENV GOPROXY https://goproxy.io

RUN mkdir -p src/order-service

WORKDIR src/order-service
COPY main.go .

RUN go mod init order-service
RUN go mod vendor
RUN go build .

EXPOSE 8889

CMD ["./order-service"]