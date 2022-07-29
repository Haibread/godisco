FROM golang:1.18-alpine as builder

RUN apk upgrade --update-cache --available
RUN apk add --no-cache \
        gcc \
        musl-dev 

RUN mkdir /app

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o godisco /app/app/godisco

FROM alpine:3.16

RUN apk --no-cache add ca-certificates

LABEL org.opencontainers.image.source="https://github.com/Haibread/godisco"

VOLUME [ "/app/config" ]
WORKDIR /app

COPY --from=builder /app/godisco /app/

ENTRYPOINT ["/app/godisco"]