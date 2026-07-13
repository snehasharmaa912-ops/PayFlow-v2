FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY go/ ./go/
WORKDIR /src/go

RUN go build -o /out/payflow ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/payflow ./payflow

EXPOSE 8080

ENTRYPOINT ["./payflow"]
