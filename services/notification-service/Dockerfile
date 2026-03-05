FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY infrastructure/ infrastructure/
COPY services/notification-service/ services/notification-service/
WORKDIR /app/services/notification-service
RUN go build -ldflags="-s -w" -o /bin/notification-service ./cmd/server/
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/notification-service /bin/app
EXPOSE 8086
USER nobody
ENTRYPOINT ["/bin/app"]
