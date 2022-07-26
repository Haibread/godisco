FROM golang:1.18-alpine as builder
WORKDIR /build
ADD . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build ./...

FROM alpine:3.16
COPY --from=builder /build/godisco /godisco
ENTRYPOINT ["/godisco"]