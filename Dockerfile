FROM golang:1.23.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git curl && \
    go install github.com/pressly/goose/v3/cmd/goose@latest

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64  

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache libc6-compat

COPY --from=builder /app/main .
COPY --from=builder /go/bin/goose /usr/local/bin/goose

EXPOSE 8080
CMD ["/app/main"]