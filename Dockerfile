FROM --platform=linux/amd64 golang:1.20.5-alpine AS builder

WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY src/ ./src/

WORKDIR /app/src
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o etl-script main.go

FROM --platform=linux/amd64 alpine:3.19
WORKDIR /app
COPY --from=builder /app/src/etl-script .

VOLUME /app/data

EXPOSE 5000


ENTRYPOINT ["./etl-script-reddit"]