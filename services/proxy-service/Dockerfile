FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY infrastructure/ infrastructure/
COPY services/proxy-service/ services/proxy-service/
WORKDIR /app/services/proxy-service
RUN go build -ldflags="-s -w" -o /bin/proxy-service ./cmd/server/
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/proxy-service /bin/app
EXPOSE 8082
USER nobody
ENTRYPOINT ["/bin/app"]
