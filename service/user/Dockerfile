FROM golang:alpine3.11

LABEL maintainer="luoman <luoman.tz@gmail.com>"

WORKDIR /go

ENV GO111MODULE on
ENV GOPROXY https://goproxy.io

RUN mkdir -p src/user-service

WORKDIR src/user-service
COPY main.go .

RUN go mod init user-service && \
    go mod vendor && \
    go build .

EXPOSE 8888

CMD ["./user-service"]