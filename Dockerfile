FROM golang:1.27-alpine AS builder

RUN mkdir /app

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o godisco /app/app/godisco

FROM alpine:3.24

RUN apk --no-cache add ca-certificates

LABEL org.opencontainers.image.source="https://github.com/Haibread/godisco"

VOLUME [ "/app/config" ]
WORKDIR /app

COPY --from=builder /app/godisco /app/

ENTRYPOINT ["/app/godisco"]
