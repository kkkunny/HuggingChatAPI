FROM golang:1.22.2-alpine3.19 AS builder

#ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY . .
RUN go build -o bin/server .


FROM alpine:3.19 AS final

RUN apk --no-cache add tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone \
ENV TZ Asia/Shanghai
RUN apk add --no-cache ca-certificates && update-ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/* /app
EXPOSE 80
ENTRYPOINT ["/app/server"]