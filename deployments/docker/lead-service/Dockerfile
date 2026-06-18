FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum go.work ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/lead-service ./cmd/lead-service

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -H appuser
WORKDIR /app
COPY --from=builder /out/lead-service /app/lead-service
USER appuser

EXPOSE 8081
ENTRYPOINT ["/app/lead-service"]

