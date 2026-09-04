FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o high-ram .

FROM alpine:3.20

COPY --from=builder /app/high-ram /usr/local/bin/high-ram

CMD ["/usr/local/bin/high-ram"]