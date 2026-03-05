FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY infrastructure/ infrastructure/
COPY services/vps-service/ services/vps-service/
WORKDIR /app/services/vps-service
RUN go build -ldflags="-s -w" -o /bin/vps-service ./cmd/server/
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/vps-service /bin/app
EXPOSE 8083
USER nobody
ENTRYPOINT ["/bin/app"]
