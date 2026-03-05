FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY infrastructure/ infrastructure/
COPY services/log-service/ services/log-service/
WORKDIR /app/services/log-service
RUN go build -ldflags="-s -w" -o /bin/log-service ./cmd/server/
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/log-service /bin/app
EXPOSE 8087
USER nobody
ENTRYPOINT ["/bin/app"]
