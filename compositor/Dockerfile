FROM golang:1.11.5-alpine3.8

WORKDIR /app

ENV GO111MODULE="on"
ENV CGO_ENABLED=0

# git
RUN apk add gcc make git openssl && rm /var/cache/apk/*

COPY go.mod go.mod

RUN go mod download

COPY compositor.go compositor.go

RUN go build -o compositor .

EXPOSE 8080
ENTRYPOINT ["./compositor"]
