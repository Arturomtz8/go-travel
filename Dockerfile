FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY src/ ./src/

WORKDIR /app/src
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o etl-script main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/src/etl-script .

ENTRYPOINT ["./etl-script"]