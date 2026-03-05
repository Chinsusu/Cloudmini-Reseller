FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY services/api-gateway/ services/api-gateway/
WORKDIR /app/services/api-gateway
RUN go build -ldflags="-s -w" -o /bin/api-gateway ./cmd/server/
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/api-gateway /bin/api-gateway
EXPOSE 8080
USER nobody
ENTRYPOINT ["/bin/api-gateway"]
