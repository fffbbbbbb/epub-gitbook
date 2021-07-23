FROM golang:alpine AS builder

WORKDIR /build

ENV GOPROXY https://goproxy.cn

COPY . .

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o main

FROM node:10-alpine

WORKDIR /build

COPY --from=builder /build /build

RUN npm install gitbook-cli -g

ENTRYPOINT ["./start.sh"]